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

package handler

import (
	"errors"
	"fmt"
	"strings"

	"bkauth/pkg/api/common"
)

const (
	// The two characters the level-pair grammar spends. Neither appears in an
	// upstream gateway, API or MCP server name, so nothing is made unreachable by
	// reserving them, but a user who types one gets a 400 -- which is why the
	// frontend is asked to strip them from the search box.
	levelPairSeparator = ","
	levelTextSeparator = ":"
)

// errMalformedLevelPairs is returned for level-pair syntax the handler reports as
// a bad request. The level names inside are not checked here: those belong to the
// realm's vocabulary, and it validates them.
var errMalformedLevelPairs = errors.New("malformed level pairs")

// grantableResourceQueryRequest is the query of the grantable resource list
// endpoint.
//
// common.PageParamSerializer is deliberately not reused. It allows a page_size
// of up to 100 while the upstream this forwards to caps a page at
// bkapigateway.MaxPageSize, so binding would accept a value the handler could
// only reject by hand.
//
// The max tag below repeats that ceiling as a literal, a struct tag being
// unable to reference a constant. It is the one copy outside the constant
// itself, so a second endpoint forwarding to the same upstream should validate
// against bkapigateway.MaxPageSize rather than spell the number a third time.
type grantableResourceQueryRequest struct {
	Type string `form:"type" binding:"required"`
	// Keyword filters one or more levels at once, written
	// "<level>:<text>,<level>:<text>" and AND-ed.
	//
	// Levels are addressed by name so that this shares the vocabulary of the
	// type's levels and of a stored grant's level. The alternative, a fixed
	// group/item pair, reintroduces exactly the positional words that were taken
	// out of those two, and leaves "the type has no such level" impossible to
	// tell from "nothing matched".
	//
	// max bounds the parsing below rather than the text a user may search for;
	// each level's own ceiling is the upstream's, applied where it is forwarded.
	Keyword string `form:"keyword" binding:"omitempty,max=512"`
	// page is bounded so the offset it multiplies out to cannot overflow. The
	// ceiling is far past any real catalog; hitting it means the caller is
	// walking pages that were never going to hold anything.
	Page     int `form:"page" binding:"omitempty,min=1,max=1000"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=20"`
}

// keywords parses Keyword into a filter per level. It returns nil when no
// keyword was sent, which is a query filtering nothing rather than one filtering
// on an empty string.
func (q *grantableResourceQueryRequest) keywords() (map[string]string, error) {
	if q.Keyword == "" {
		return nil, nil
	}
	return parseLevelPairs(q.Keyword)
}

// grantableResourceRefRequest is the query of the grantable resource lookup
// endpoint, which names one entry instead of searching for it. It is what makes a
// resource the catalog does not list grantable at all: the listing upstream
// returns public objects only, so a private MCP server or API is reached by being
// named.
//
// Name repeats the listing endpoint's "<level>:<text>,<level>:<text>" grammar and
// the same 512-byte ceiling, so a client speaks one syntax for both. The
// difference is in the meaning, not the shape: here each text is an exact name,
// and the pairs together must identify a single entry rather than narrow a page.
//
// It is required, unlike keyword. A lookup with nothing to look up has no sensible
// answer, where a listing without a filter simply returns everything.
type grantableResourceRefRequest struct {
	Type string `form:"type" binding:"required"`
	Name string `form:"name" binding:"required,max=512"`
}

func (q *grantableResourceRefRequest) names() (map[string]string, error) {
	return parseLevelPairs(q.Name)
}

// parseLevelPairs splits "<level>:<text>,<level>:<text>" into one text per level.
//
// Every malformed case is an error rather than something skipped. A dropped pair
// would widen a search silently, and answering a narrowed query with more rows
// than asked for is worse than answering with a 400; on the lookup path a dropped
// pair is worse still, since the pairs are what single out the entry.
func parseLevelPairs(raw string) (map[string]string, error) {
	parsed := make(map[string]string)
	for _, pair := range strings.Split(raw, levelPairSeparator) {
		level, text, found := strings.Cut(pair, levelTextSeparator)
		switch {
		case !found:
			return nil, fmt.Errorf(
				"%w: %q is missing the %q that separates its level from its text",
				errMalformedLevelPairs, pair, levelTextSeparator)
		case level == "":
			return nil, fmt.Errorf("%w: %q names no level", errMalformedLevelPairs, pair)
		case text == "":
			// Distinguished from an absent level so that a form wired up wrong
			// fails loudly instead of quietly returning everything.
			return nil, fmt.Errorf("%w: %q has no text to match", errMalformedLevelPairs, pair)
		}

		if _, duplicated := parsed[level]; duplicated {
			// Two texts for one level cannot both be honoured, and picking either
			// would drop the other without saying so.
			return nil, fmt.Errorf(
				"%w: level %q is given twice", errMalformedLevelPairs, level)
		}
		parsed[level] = text
	}

	return parsed, nil
}

func (q *grantableResourceQueryRequest) limit() int {
	if q.PageSize == 0 {
		return common.DefaultPageSize
	}
	return q.PageSize
}

func (q *grantableResourceQueryRequest) offset() int {
	page := q.Page
	if page == 0 {
		page = 1
	}
	return (page - 1) * q.limit()
}
