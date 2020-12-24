#!/bin/sh
set -e -x
CONFIG_FILE="/tmp/configs/delete_before_permanent_delta_test_config.json"

COMMON_CONFIG="/tmp/configs/common_config.json"
TMP_CONFIG="/tmp/configs/tmp_config.json"
cat ${CONFIG_FILE} > ${TMP_CONFIG}
echo "," >> ${TMP_CONFIG}
cat ${COMMON_CONFIG} >> ${TMP_CONFIG}
/tmp/scripts/wrap_config_file.sh ${TMP_CONFIG}

/usr/lib/postgresql/10/bin/initdb ${PGDATA}

echo "archive_mode = on" >> /var/lib/postgresql/10/main/postgresql.conf
echo "archive_command = 'WALG_DELTA_MAX_STEPS=3 \
/usr/bin/timeout 600 /usr/bin/wal-g --config=${TMP_CONFIG} wal-push %p'" >> /var/lib/postgresql/10/main/postgresql.conf
echo "archive_timeout = 600" >> /var/lib/postgresql/10/main/postgresql.conf

/usr/lib/postgresql/10/bin/pg_ctl -D ${PGDATA} -w start

/tmp/scripts/wait_while_pg_not_ready.sh

#delete all backups of any
WALG_DELTA_MAX_STEPS=3 wal-g --config=${TMP_CONFIG} delete everything FORCE --confirm --use-sentinel-time

# push permanent and impermanent delta backups
for i in 1 2 3 4
do
    pgbench -i -s 1 postgres &
    sleep 1
    if [ $i -eq 3 ]
    then
        WALG_DELTA_MAX_STEPS=3 wal-g --config=${TMP_CONFIG} backup-push --permanent ${PGDATA}
        pg_dumpall -f /tmp/dump1
    else
        WALG_DELTA_MAX_STEPS=3 wal-g --config=${TMP_CONFIG} backup-push ${PGDATA}
    fi
done

WALG_DELTA_MAX_STEPS=3 wal-g --config=${TMP_CONFIG} backup-list --detail

# delete backups by pushing a full backup and running `delete retain 1`
# this should only delete the last impermanent delta backup
pgbench -i -s 1 postgres &
sleep 1
WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} backup-push ${PGDATA}

WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} backup-list --detail

WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} delete retain 1 --confirm --use-sentinel-time

WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} backup-list

# restore the backup and compare with previous state
/tmp/scripts/drop_pg.sh
first_backup_name=`\
WALG_DELTA_MAX_STEPS=0 \
wal-g --config=${TMP_CONFIG} backup-list | sed '2q;d' | cut -f 1 -d " "`

WALG_DELTA_MAX_STEPS=0 \
wal-g --config=${TMP_CONFIG} backup-fetch ${PGDATA} $first_backup_name

echo "restore_command = 'echo \"WAL file restoration: %f, %p\"&& \
WALG_DELTA_MAX_STEPS=0 /usr/bin/wal-g --config=${TMP_CONFIG} wal-fetch \"%f\" \"%p\"'" > ${PGDATA}/recovery.conf
/usr/lib/postgresql/10/bin/pg_ctl -D ${PGDATA} -w start
/tmp/scripts/wait_while_pg_not_ready.sh
pg_dumpall -f /tmp/dump2
diff /tmp/dump1 /tmp/dump2

WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} backup-list --detail

# delete all backups after previous base_tests
WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} delete everything FORCE --confirm --use-sentinel-time

# make impermanent base backup
WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} backup-push ${PGDATA}

imperm_backup=`WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} backup-list | egrep -o "[0-9A-F]{24}"`

# make permanent base backup
WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} backup-push --permanent ${PGDATA}

WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} backup-list --detail

# check that nothing changed when permanent backups exist
WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} backup-list > /tmp/dump1

WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} delete everything --confirm --use-sentinel-time || true

WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} backup-list > /tmp/dump2
diff /tmp/dump1 /tmp/dump2

rm /tmp/dump2
touch /tmp/dump2

# delete all backups
WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} delete everything FORCE --confirm --use-sentinel-time

WALG_DELTA_MAX_STEPS=0 wal-g --config=${TMP_CONFIG} backup-list 2> /tmp/2 1> /tmp/1

# check that stdout not include any backup
! cat /tmp/1 | egrep -o "[0-9A-F]{24}" > /tmp/dump1
diff /tmp/dump1 /tmp/dump2

# check that stderr not include any backup
# stderr shuld be "INFO: ... No backups found"
! cat /tmp/2 | egrep -o "[0-9A-F]{24}" > /tmp/dump1
diff /tmp/dump1 /tmp/dump2

/tmp/scripts/drop_pg.sh

