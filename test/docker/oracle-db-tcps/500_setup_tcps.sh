#!/usr/bin/env bash
# Sets up a TCPS listener with an orapki auto-login wallet, the way a real
# Oracle DBA would configure one, so tests can connect via TCPS/WALLET the
# same way sqlplus does. Needs the "-full" image (Java/orapki, unlike -slim).
set -e

TNS_ADMIN=${TNS_ADMIN:-$ORACLE_HOME/network/admin}
WDIR="$TNS_ADMIN/wallet"
WPASS="TcpsTestWallet2026#"
CLIENT_WALLET_DIR=/client-wallet

mkdir -p "$WDIR"
orapki wallet create -wallet "$WDIR" -pwd "$WPASS" -auto_login
orapki wallet add -wallet "$WDIR" -pwd "$WPASS" -dn "CN=tnscli-oracledb" -keysize 2048 -self_signed -validity 3650

cat >"$TNS_ADMIN/listener.ora" <<EOF
LISTENER =
  (DESCRIPTION_LIST =
    (DESCRIPTION =
      (ADDRESS = (PROTOCOL = IPC)(KEY = EXTPROC_FOR_FREE))
      (ADDRESS = (PROTOCOL = TCP)(HOST=0.0.0.0)(PORT = 1521))
      (ADDRESS = (PROTOCOL = TCPS)(HOST=0.0.0.0)(PORT = 2484))
    )
  )
DEFAULT_SERVICE_LISTENER = FREE
SSL_CLIENT_AUTHENTICATION = FALSE
WALLET_LOCATION =
  (SOURCE =
    (METHOD = FILE)
    (METHOD_DATA =
      (DIRECTORY = $WDIR)
    )
  )
EOF

cat >>"$TNS_ADMIN/sqlnet.ora" <<EOF
WALLET_LOCATION =
  (SOURCE =
    (METHOD = FILE)
    (METHOD_DATA =
      (DIRECTORY = $WDIR)
    )
  )
SSL_CLIENT_AUTHENTICATION = FALSE
EOF

lsnrctl stop || true
lsnrctl start
sleep 3

# force service registration instead of waiting for PMON's next interval
sqlplus -s / as sysdba <<EOF
alter system register;
EOF
sleep 3
lsnrctl status

if [ -d "$CLIENT_WALLET_DIR" ]; then
  # best effort: a permission hiccup on the bind-mounted host dir must not
  # abort container init (the DB itself is already up and fine at this point)
  cp -r "$WDIR"/. "$CLIENT_WALLET_DIR"/ || echo "warning: could not copy wallet to $CLIENT_WALLET_DIR"
  # orapki writes wallet files as 0600 owned by the container user; open them
  # up so the host-side test process (a different uid) can read them back
  chmod -R a+r "$CLIENT_WALLET_DIR" || true
fi
