#!/bin/sh
test -f /secrets/app.env && source /secrets/app.env
test -f /secrets/extra.env && source /secrets/extra.env

exec "$@"
