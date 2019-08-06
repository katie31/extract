#!/bin/sh
set -e -x
CONFIG_FILE="/tmp/configs/several_delta_backups_test_config.json"
COMMON_CONFIG="/tmp/configs/common_config.json"
echo "," >> ${CONFIG_FILE}
cat ${COMMON_CONFIG} >> ${CONFIG_FILE}

tmp/scripts/wrap_config_file.sh ${CONFIG_FILE}

/usr/lib/postgresql/10/bin/initdb ${PGDATA}

echo "archive_mode = on" >> /var/lib/postgresql/10/main/postgresql.conf
echo "archive_command = '/usr/bin/timeout 600 /usr/bin/wal-g --config=${CONFIG_FILE} wal-push %p && mkdir -p /tmp/deltas/$(basename %p)'" >> /var/lib/postgresql/10/main/postgresql.conf
echo "archive_timeout = 600" >> /var/lib/postgresql/10/main/postgresql.conf

/usr/lib/postgresql/10/bin/pg_ctl -D ${PGDATA} -w start

pgbench -i -s 10 postgres
pgbench -T 100000000 postgres &
wal-g --config=${CONFIG_FILE} backup-push ${PGDATA}

wal-g --config=${CONFIG_FILE} backup-push ${PGDATA}

psql -f scripts/amcheck.sql -v "ON_ERROR_STOP=1" postgres

wal-g --config=${CONFIG_FILE} backup-push ${PGDATA}

wal-g --config=${CONFIG_FILE} backup-push ${PGDATA}

pkill pgbench

/usr/lib/postgresql/10/bin/pg_ctl -D ${PGDATA} -m smart -w stop
/usr/lib/postgresql/10/bin/pg_ctl -D ${PGDATA} -w start

pg_dumpall -f /tmp/dump1
sleep 1

scripts/drop_pg.sh

wal-g --config=${CONFIG_FILE} backup-fetch ${PGDATA} LATEST

echo "restore_command = 'echo \"WAL file restoration: %f, %p\"&& /usr/bin/wal-g --config=${CONFIG_FILE} wal-fetch \"%f\" \"%p\"'" > ${PGDATA}/recovery.conf

/usr/lib/postgresql/10/bin/pg_ctl -D ${PGDATA} -w start
sleep 10


pg_dumpall -f /tmp/dump2

psql -f scripts/amcheck.sql -v "ON_ERROR_STOP=1" postgres

diff /tmp/dump1 /tmp/dump2

scripts/drop_pg.sh

echo "Several delta backup success!!!!!!"
