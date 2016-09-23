#!/bin/sh
if [ "${1:0:1}" = '-' ]; then
  set -- depman-srv "$@"
fi

if [ "$1" = 'depman-srv' ]; then
  /depman-srv -l $LISTEN -n $NAMESPACE -d $LOGLEVEL -s $STOREDIR
fi

exec "$@"
