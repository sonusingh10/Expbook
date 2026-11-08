#!/usr/bin/env sh
# Run Expbook locally from source (SQLite). Builds the backend/frontend if missing.
set -e
cd "$(dirname "$0")"

if [ ! -f .env ]; then
    cp .env.example .env
    secret=$(openssl rand -hex 32)
    sed -i.bak "s/^EBK_SECURITY_SECRET_KEY=.*/EBK_SECURITY_SECRET_KEY=${secret}/" .env && rm -f .env.bak
    echo "Created .env with a random secret key."
fi

mkdir -p data storage log

[ -x ./ezbookkeeping ] || ./build.sh backend --no-lint --no-test
[ -d ./dist ] || ./build.sh frontend --no-lint --no-test

set -a
. ./.env
set +a
exec ./ezbookkeeping server run
