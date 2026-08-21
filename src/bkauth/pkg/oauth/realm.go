/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth 服务 (BlueKing - Auth) available.
 * Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *     http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

const (
	// AudienceLevelNone is the level of a grant that came from no level: the
	// type-wide box above the catalog rather than a row in it. Empty for the same
	// reason an unselectable row has an empty Audience -- in this protocol an
	// empty string is how a field says it does not apply.
	//
	// Spelled out rather than written as "" in a struct literal, where it would
	// be indistinguishable from a field nobody filled in.
	AudienceLevelNone = ""
)

var (
	// realms is written during single-threaded startup (initRealms) and read-only
	// after the HTTP server starts, so no mutex is needed.
	realms = make(map[string]Realm)

	// ErrUnknownGrantableResourceType is returned by ListGrantableResource when
	// the query names a type the realm does not serve. It is a sentinel so the
	// web layer can answer 400 instead of 500: the caller asked for something
	// that does not exist rather than tripping over a fault.
	ErrUnknownGrantableResourceType = errors.New("unknown grantable resource type")

	// ErrUnknownGrantableResourceLevel is returned when a query keyword names a
	// level the type does not have. Same reasoning as the type sentinel: the
	// caller named something that does not exist.
	//
	// It answers rather than ignores because the alternative is a page of rows
	// that plainly do not match what was typed, or an empty page for a filter
	// that was never applied -- both read as bugs to whoever is looking at the
	// screen.
	ErrUnknownGrantableResourceLevel = errors.New("unknown grantable resource level")

	// ErrGrantableResourceNotFound is returned by ResolveGrantableResource when
	// the realm has nothing under the names given. A sentinel so the web layer
	// answers 404 rather than 500, and distinct from the two above: those say the
	// caller used a word the realm does not know, this says the words were fine
	// and named nothing.
	ErrGrantableResourceNotFound = errors.New("grantable resource not found")

	// ErrIncompleteGrantableResourceRef is returned when a ref leaves out a level
	// the type needs in order to name a single entry.
	//
	// Separate from the not-found sentinel because the two lead somewhere
	// different: a missing level is a form the user has not finished filling in,
	// while not-found is a name to go back and check. Reporting the former as the
	// latter would tell someone their gateway name is wrong when they simply have
	// not typed the MCP server yet.
	ErrIncompleteGrantableResourceRef = errors.New("incomplete grantable resource ref")

	// ErrGrantableResourceNotGrantable is returned when the names given do
	// identify an object, but one this kind of token is not allowed to be
	// granted -- its owner has not opened it to personal access tokens.
	//
	// Kept apart from the not-found sentinel because the two lead somewhere
	// different, the same way not-found and incomplete do: not-found is a name to
	// go back and check, this is a name spelled exactly right whose owner has to
	// change a setting. Folding it into not-found would send the user hunting for
	// a typo that is not there.
	//
	// Only realms whose catalog is filtered upstream need it. Where the filter
	// and the resolve read the same in-memory set there is nothing to diverge:
	// anything resolvable is grantable by construction.
	ErrGrantableResourceNotGrantable = errors.New("grantable resource not grantable")
)

// Extras carries display-only facts about the row it hangs on, such as
// is_official. A key is present only when its value is known, so an absent key
// means "unknown" and must not be read as false. Nothing in here may drive
// selection or permission logic.
//
// It is always serialized, as {} when nothing is known rather than omitted, so a
// reader asks which keys are present and never whether the object is. That is
// also what keeps attaching a fact later a change to one level's key set rather
// than a change to whether that level carries extras at all: a client written
// against a row that has none keeps working when it gains one.
type Extras map[string]any

// MarshalJSON renders a nil Extras as {} rather than null.
//
// The invariant lives on the type rather than at every construction site because
// the Go default for a nil map is null: a realm that simply leaves the field
// unset would otherwise emit a different JSON type for "nothing known" than for
// "something known", and holding the guarantee would be one forgotten map
// literal away.
func (e Extras) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("{}"), nil
	}
	// Converted to the unnamed type, which does not have this method and so
	// cannot recurse back into it.
	return json.Marshal(map[string]any(e))
}

// Realm defines per-realm resource handling strategy.
// Implementations live in pkg/realm/{name}/ and are registered at startup.
//
// The resource and the audience direction are declared separately even though
// today every realm happens to spell the two alike. The OAuth path is handed a
// resource, which the client asks for coarsely and consent narrows into
// audiences; the personal access token path is handed audiences the user picked
// outright. The two inputs come from different places, so a realm is free to
// let them diverge without any caller silently doing the wrong thing.
type Realm interface {
	Name() string

	ValidateResource(ctx context.Context, resource string) error
	ExtractAudiences(ctx context.Context, resource string) ([]string, error)
	ResolveResourceDisplay(ctx context.Context, resource string) (any, error)

	// ValidateAudiences reports whether every entry is a token this realm is
	// willing to store. It is purely syntactic and makes no upstream call, so a
	// personal access token can still be written while the gateway is down.
	ValidateAudiences(ctx context.Context, audiences []string) error

	// ResolveAudienceDisplays renders the given tokens, keyed by the token each
	// rendering is of. tenantID scopes the upstream lookups; leaving it empty
	// yields only tenant-neutral data.
	//
	// It takes many tokens at once rather than one at a time so a realm can
	// collapse the upstream lookups they have in common. A list page holds ten
	// personal access tokens that mostly name the same handful of gateways, and
	// resolving them one by one multiplies a handful of calls into dozens of
	// serial ones.
	//
	// The result is keyed rather than positional because a rendering is a
	// function of the token alone -- see AudienceDisplay -- so which grant list a
	// token appeared in cannot change what it renders as. That leaves the caller
	// free to pass the tokens of a whole page flat and read the renderings back
	// per grant list, in whatever order it stores them, and it is what keeps this
	// method from having to carry the caller's grouping through work that starts
	// by discarding it. A token repeated across the input is rendered once.
	//
	// A key is present only when the realm rendered that token, so a caller must
	// tolerate an absent one and fall back to the token itself.
	//
	// Upstream failures degrade to bare names rather than erroring: a token page
	// that renders without titles is far better than one that will not render. A
	// token this realm cannot parse is an error, and it is the whole call's:
	// stored tokens are validated on write, so an unparseable one means the realm
	// and the stored data disagree rather than that one row is odd.
	ResolveAudienceDisplays(
		ctx context.Context, tenantID string, audiences []string,
	) (map[string]AudienceDisplay, error)

	// GrantableResourceTypes returns the types the realm's grantable resources
	// are divided into, in display order. It is static metadata and takes no
	// query.
	//
	// It also supplies the label for AudienceDisplay.Type, so a client rendering
	// stored tokens needs it too, not just the picker.
	GrantableResourceTypes() []GrantableResourceType

	// ListGrantableResource enumerates what a token in this realm may be
	// granted. Unlike the audience methods above it is not required for
	// correctness -- a realm serving nothing here still stores and validates
	// tokens fine -- but it is required all the same, so that the frontend has
	// one code path instead of a hard-coded table per realm it must keep in step
	// with us. A realm whose set is two compiled-in constants answers from those
	// constants; that is a handful of lines against a table maintained in
	// another repository.
	//
	// Note what this does not carry: how many entries a valid selection holds.
	// That lives in ValidateAudiences alone, so a caller learns it by being
	// rejected. Realms differ there (one demands exactly one entry), and until
	// the constraint is expressed here the frontend cannot derive it.
	//
	// tenantID scopes the upstream lookups, on the same terms as
	// ResolveAudienceDisplays: it says who is asking rather than what is asked
	// for, which is why it sits beside ctx instead of in the query.
	ListGrantableResource(
		ctx context.Context, tenantID string, q GrantableResourceQuery,
	) (GrantableResourcePage, error)

	// ResolveGrantableResource names one grantable entry exactly, rather than
	// searching for it.
	//
	// It exists because the catalog cannot enumerate everything a token may be
	// granted: blueking's is served by upstream endpoints that return public
	// objects only, so a private MCP server or API is grantable but never listed.
	// The user types its name instead, and only the realm can turn that into an
	// audience -- the token grammar is the one thing clients are told not to
	// build. What comes back is a catalog row, the same type ListGrantableResource
	// returns, so a caller treats a typed-in entry exactly like a ticked one.
	//
	// This is where existence is checked, and it is the only place. Write-time
	// validation stays syntactic (see ValidateAudiences), which is what keeps a
	// token editable while the upstream is down and keeps a grant on a since-
	// deleted object from blocking a rename.
	//
	// Existence is not the whole of it: an implementation must also refuse an
	// object that exists but is closed to personal access tokens
	// (ErrGrantableResourceNotGrantable). ListGrantableResource gets that filter
	// for free where the upstream applies it, and this path does not -- naming an
	// object outright is exactly what skips the catalog's filtering.
	//
	// Unlike ResolveAudienceDisplays an upstream failure is an error rather than a
	// degraded answer: confirming the entry is the whole point of the call, so
	// answering without having confirmed it would report "found" for something
	// nobody looked up.
	ResolveGrantableResource(
		ctx context.Context, tenantID string, ref GrantableResourceRef,
	) (GrantableResource, error)
}

// AudienceDisplay is one stored audience rendered for display. It is a function
// of that audience alone: the same token renders the same way whichever grant
// list it sits in, which is what lets ResolveAudienceDisplays answer for a whole
// page at once and be keyed by token, and what lets a caller relate a rendering
// back to the grant it came from.
//
// It is flat rather than the grouped tree the consent page renders. A tree reads
// better, but it cannot be walked back to the audiences behind it without
// knowing the realm's token syntax -- the one thing the frontend is told not to
// parse. Naming, removing or highlighting a single grant needs that mapping, and
// only the backend can produce it without parsing. Grouping the entries back up
// is left to the client, which has Type and Level to do it with.
//
// An entry is the catalog row the grant was picked from, said back: its type,
// name, display name, audience and extras are that row's, field for field. So a
// client needs one vocabulary rather than one per direction, and correlating the
// two is comparing audience against audience, both of them the token.
type AudienceDisplay struct {
	// Type is which division of the catalog this is, labelled by the matching
	// GrantableResourceType rather than by a copy on every entry.
	Type string `json:"type"`

	// Level is the level of the catalog this grant was picked at, naming one
	// entry of the type's Levels, or AudienceLevelNone for a grant picked above
	// the catalog altogether.
	//
	// It is the one thing an entry cannot be read back from: a client cannot tell
	// a gateway grant from an API grant by looking, and cannot look it up either,
	// the catalog being paged -- the gateway a granted API belongs to may be
	// pages away from the one on screen. A reader groups by it to show grants the
	// way they were picked.
	Level string `json:"level"`

	// Name is the row's bare identifier: the gateway for a grant at the gateway
	// level, the MCP server or API at the level below, and "*" for a type-wide
	// grant, which has no identifier of its own.
	Name string `json:"name"`

	DisplayName string `json:"display_name"`

	// Audience is the token verbatim, equal to one entry of the stored audience
	// array and to the Audience of the catalog row this grant came from.
	Audience string `json:"audience"`

	// Extras carries display-only facts about this grant; see the type for the
	// contract it comes with.
	//
	// Its keys are not the catalog row's, so the two directions must not be
	// compared key for key. A fact describing the level above repeats on every
	// entry under it -- is_official describes the gateway, and an API grant has
	// no gateway row of its own to hang it on -- and is_public appears here and on
	// a resolved row, but never while browsing, that catalog's upstream returning
	// public objects exclusively.
	Extras Extras `json:"extras"`
}

// The four types below carry what the grantable resource pair enumerates.
//
// "Grantable" is the load-bearing word: this is what a token *could* be given, a
// different set from what any particular token already holds. The two are
// siblings in the API, so a name that did not distinguish them -- a bare
// "resources" -- would read as the latter.

// GrantableResourceType is one division of a realm's grantable resources. For
// blueking the two are MCP servers and gateway APIs, which are also two distinct
// upstream endpoints rather than a presentational split.
//
// Name is the code every payload referring to a type identifies it by -- the
// consent page's ResourceGroup.Type, AudienceDisplay.Type, and the type query of
// ListGrantableResource. It is spelled "name" here and "type" there because a
// type names itself but is referred to by its relation, the same way a row names
// itself with GrantableResource.Name.
//
// DisplayName is the type's only label, so no other payload carries a copy.
type GrantableResourceType struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	// Audience grants the type as a whole, including members created later.
	// Same contract as GrantableResource.Audience: the token this contributes
	// when selected, empty when the type has no such token and the frontend
	// should not offer a select-all box.
	Audience string `json:"audience"`
	// Levels describes how deep this type's catalog nests, outermost first, and
	// labels each level. One entry means a flat list, so a client knows before
	// fetching a page whether to render an expander. The names are also the only
	// keys GrantableResourceQuery.Keywords accepts for this type.
	//
	// It says nothing about what can be selected: that is per row, and only the
	// row's own Audience says it. So a level may be listed here and never appear
	// in AudienceDisplay.Level -- the gateways grouping MCP servers being the
	// case, grantable as they are not.
	Levels []GrantableResourceLevel `json:"levels"`
}

// GrantableResourceLevel labels one level of a type's catalog.
//
// The names are the realm's to choose, and are expected to read as what the
// level holds -- "gateway", "api" -- rather than as a position in the tree.
// Position is already carried by the order of GrantableResourceType.Levels, and
// a name like "group" would leave a reader comparing it against its own label
// for a correspondence that is not there.
type GrantableResourceLevel struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// GrantableResourceQuery speaks limit/offset because that is what the upstream
// takes. The page/page_size pair the web layer exposes is converted once, in
// the handler, rather than in every realm.
//
// It holds what is being asked for and nothing about who is asking; the tenant
// is a parameter of ListGrantableResource for that reason.
type GrantableResourceQuery struct {
	Type string
	// Keywords filters the catalog one level at a time, keyed by the level's
	// GrantableResourceLevel.Name. Entries are AND-ed, matching the upstream,
	// which AND-s its own gateway and item name filters.
	//
	// Keyed by level rather than by a fixed group/item pair so that the levels a
	// type declares, the level a stored grant reports, and the level a filter
	// names are one vocabulary instead of three. It also makes naming a level
	// the type does not have an error the realm can see, where a positional pair
	// left it indistinguishable from a filter that simply matched nothing.
	//
	// Validate it with ValidateLevelNames before reading, then read by level
	// name; a missing key is a level nobody filtered on.
	Keywords map[string]string

	Limit  int
	Offset int
}

// GrantableResourceRef names a single entry of one type, for the times a caller
// already knows what it wants instead of browsing for it.
//
// Names is keyed by level, the same vocabulary GrantableResourceQuery.Keywords
// speaks, and differs from it in two ways: the values are matched exactly rather
// than as substrings, and together they must pin down one entry. Which levels
// that takes is the type's business -- blueking needs both of its, so a gateway
// with no MCP server named is ErrIncompleteGrantableResourceRef rather than a
// stand-in for every server of that gateway.
type GrantableResourceRef struct {
	Type string
	// Names holds one exact name per level; validate it with ValidateLevelNames
	// before reading, then read by level name.
	Names map[string]string
}

// ValidateLevelNames reports whether every key names one of the levels given,
// which are the levels of the type being queried or referred to.
//
// It serves both GrantableResourceQuery.Keywords and GrantableResourceRef.Names:
// the two hold different things -- a substring to filter on, an exact name to
// resolve -- but they are keyed by the same level vocabulary, and a caller
// aiming at a level the type does not have is the same mistake either way.
//
// Realms call it rather than the web layer: a level name belongs to the same
// vocabulary as the type name, and the handler would have to rebuild the type to
// levels mapping to check it, which is a second copy of what the realm already
// declares in GrantableResourceTypes.
func ValidateLevelNames(named map[string]string, levels ...string) error {
	for level := range named {
		if !slices.Contains(levels, level) {
			return fmt.Errorf("%w: %q", ErrUnknownGrantableResourceLevel, level)
		}
	}
	return nil
}

// GrantableResource is one entry: either something a token can be granted, or a
// grouping row holding such entries.
type GrantableResource struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	// Extras carries display-only facts such as is_official; see the type for the
	// contract it comes with.
	//
	// Which keys appear depends on where the row came from rather than on its
	// level. Browsing the catalog, only group rows carry anything, the facts on
	// offer describing gateways, so an item row serializes it as {}. A row from
	// ResolveGrantableResource carries is_public, which the browsable catalog has
	// no use for: its upstream returns public objects exclusively, and a private
	// one is reachable only by being named.
	Extras Extras `json:"extras"`
	// Audience is the token this entry contributes when granted, and is opaque
	// to the frontend: it is collected, submitted as-is, and matched by equality
	// when refilling the form. Empty means the entry is only a grouping handle
	// and cannot be granted on its own.
	Audience string `json:"audience"`
	// Items reuses the field name the consent page already ships so one
	// component can render both trees. Leaf entries omit it, which is what makes
	// the depth self-describing.
	Items []GrantableResource `json:"items,omitempty"`
}

// GrantableResourcePage is one page of a realm's grantable resources.
type GrantableResourcePage struct {
	// Count is the total in the realm's own pagination unit, which for blueking
	// is gateways rather than grantable entries. Clients paging by entry count
	// will compute the wrong number of pages.
	Count   int                 `json:"count"`
	Results []GrantableResource `json:"results"`
}

// RegisterRealm registers a Realm implementation. Must be called during startup
// before the HTTP server begins accepting requests.
func RegisterRealm(r Realm) {
	realms[r.Name()] = r
}

// GetRealm returns the Realm implementation for the given name, or nil if not found.
func GetRealm(name string) Realm {
	return realms[name]
}

// IsValidRealm reports whether the given name corresponds to a registered realm.
func IsValidRealm(name string) bool {
	_, ok := realms[name]
	return ok
}

// PageFlatGrantableResources filters and pages entries a realm holds in memory.
// It serves realms whose grantable set is compiled in and has one level, which
// is every realm but blueking. level names that level, so a keyword aimed
// anywhere else is rejected here rather than in each such realm.
//
// It also exists so the decision below is made once rather than argued again:
// Limit and Offset are honoured even though these sets hold a handful of
// entries. Returning everything regardless is cheap and works, right up to the
// point where a client trusts Count to page and silently sees the first entries
// on every page.
//
// Matching is over both Name and DisplayName, which is more than blueking can
// offer -- its filtering happens upstream, by name only. The contract is
// deliberately per-realm rather than levelled down to the strictest: here both
// are free, and searching gpu's "所有" would otherwise match nothing a user can
// see.
func PageFlatGrantableResources(
	all []GrantableResource,
	level string,
	q GrantableResourceQuery,
) (GrantableResourcePage, error) {
	if err := ValidateLevelNames(q.Keywords, level); err != nil {
		return GrantableResourcePage{}, err
	}

	// Allocated with a zero length throughout so an empty page serializes as []
	// rather than null.
	keyword := strings.ToLower(q.Keywords[level])
	matched := make([]GrantableResource, 0, len(all))
	for _, entry := range all {
		if keyword == "" ||
			strings.Contains(strings.ToLower(entry.Name), keyword) ||
			strings.Contains(strings.ToLower(entry.DisplayName), keyword) {
			matched = append(matched, entry)
		}
	}

	// Count is the total before paging, so it is taken before the slicing below.
	count := len(matched)
	if q.Offset > 0 {
		if q.Offset >= count {
			return GrantableResourcePage{Count: count, Results: []GrantableResource{}}, nil
		}
		matched = matched[q.Offset:]
	}
	if q.Limit > 0 && q.Limit < len(matched) {
		matched = matched[:q.Limit]
	}

	return GrantableResourcePage{Count: count, Results: matched}, nil
}

// FindFlatGrantableResource picks the entry a ref names out of a set the realm
// holds in memory. It is the ResolveGrantableResource counterpart of
// PageFlatGrantableResources and serves the same realms: those whose grantable
// set is compiled in and one level deep.
//
// Matching is on Name alone, unlike the paging helper, which also matches
// DisplayName. A ref names an entry rather than describing it, and Name is the
// only field that identifies one -- two entries may well read the same to a
// human, and picking whichever came first would be a coin toss.
func FindFlatGrantableResource(
	all []GrantableResource,
	level string,
	ref GrantableResourceRef,
) (GrantableResource, error) {
	if err := ValidateLevelNames(ref.Names, level); err != nil {
		return GrantableResource{}, err
	}

	name := ref.Names[level]
	if name == "" {
		return GrantableResource{}, fmt.Errorf(
			"%w: %q must be named", ErrIncompleteGrantableResourceRef, level)
	}

	for _, entry := range all {
		if entry.Name == name {
			return entry, nil
		}
	}

	return GrantableResource{}, fmt.Errorf(
		"%w: no %s named %q", ErrGrantableResourceNotFound, level, name)
}
