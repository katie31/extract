#!/bin/sh
set -e -x

CONFIG_FILE="/tmp/configs/wal_perftest_config.json"

COMMON_CONFIG="/tmp/configs/common_config.json"
TMP_CONFIG="/tmp/configs/tmp_config.json"
cat ${CONFIG_FILE} > ${TMP_CONFIG}

echo "," >> ${TMP_CONFIG}
cat ${COMMON_CONFIG} >> ${TMP_CONFIG}

tmp/scripts/wrap_config_file.sh ${TMP_CONFIG}

WAL_PUSH_LOGS="/tmp/logs/wal_push_logs/pg_wal_perftest_logs"
WAL_FETCH_LOGS="/tmp/logs/wal_fetch_logs/pg_wal_perftest_logs"

/usr/lib/postgresql/10/bin/initdb ${PGDATA}
/usr/lib/postgresql/10/bin/pg_ctl -D ${PGDATA} -w start
pgbench -i -s 50 postgres
du -hs ${PGDATA}
sleep 1
WAL=`ls -l ${PGDATA}/pg_wal | head -n2 | tail -n1 | egrep -o "[0-9A-F]{24}"`

du -hs ${PGDATA}
/usr/bin/time -v -a --output ${WAL_PUSH_LOGS} wal-g --config=${TMP_CONFIG} wal-push ${PGDATA}/pg_wal/${WAL}
sleep 1
tmp/scripts/drop_pg.sh

/usr/bin/time -v -a --output ${WAL_FETCH_LOGS} wal-g --config=${TMP_CONFIG} wal-fetch ${WAL} ${PGDATA}
sleep 1
tmp/scripts/drop_pg.sh

echo "Wal perftest success"
