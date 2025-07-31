# bkauth

This project provides some APIs for  authentication.

note:

- depends on database only
- use cache, should be fast

## layers

> view(api/*/*.go) -> service -> cache -> dao -> database

- `view` only do the validation and conversion
- `service` do the business logic
- `cache` only care about the cache
- `dao` do the query
- `database` is the database of bkauth

## develop

- `go 1.23` required

build and run

```bash
# install tools
make init

# download vendor
make dep

# build
make build

# build and serve
make serve

```

develop and test

```bash
# test
make test

# generate mock files
make mock


# do format
make fmt

# check lint
make lint

# build image
make docker-build
```

## api testing

we use [bruno](https://www.usebruno.com/) as the api testing tool.

1. import `src/bkauth/test/bkauth` into bruno
2. add `cp src/bkauth/test/dev.bru.tpl src/bkauth/test/dev.bru` and update the host
    - the `x_bk_app_code: bk_paas3` and `x_bk_app_secret: G3dsdftR9nGQM8WnF1qwjGSVE0ScXrz1hKWM` should be the same in the config.yaml
3. switch the env into dev and test all apis
