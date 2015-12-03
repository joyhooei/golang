#!/bin/bash
#usage: ./script_name {start|stop|status|restart}
# Source function library.
. /etc/rc.d/init.d/functions
# Source networking configuration.
. /etc/sysconfig/network
# Check networking is up.
[ "$NETWORKING" = "no" ] && exit 0
RETVAL=0
PID=
FILTER_DIR="/usr/local/word_filter"
FILTER="${FILTER_DIR}/bin/word_filter"
PROG=$(basename $FILTER)
CONF="${FILTER_DIR}/conf/${PROG}.conf"
if [ ! -f $CONF ]; then
    echo -n $"$CONF not exist.";warning;echo
    exit 1
fi
PID_FILE=${FILTER_DIR}/word_filter.pid
if [ ! -x $FILTER ]; then
    echo -n $"$FILTER not exist.";warning;echo
    exit 0
fi

start() {
    echo -n $"Starting $PROG: "
    nohup $FILTER $CONF 2>&1 >/dev/null &
    RETVAL=$?
    if [ $RETVAL -eq 0 ]; then
        success;echo
	PID=$(pidof $FILTER)
	echo $PID > $PID_FILE
    else
        failure;echo
    fi
    return $RETVAL
}
stop() {
    echo -n $"Stopping $PROG: "
    if [ -f $PID_FILE ] ;then
      read PID <  "$PID_FILE" 
    else 
      failure;echo;
      echo -n $"$PID_FILE not found.";failure;echo
      return 1;
    fi
    if checkpid $PID; then
    kill -TERM $PID >/dev/null 2>&1
        RETVAL=$?
        if [ $RETVAL -eq 0 ] ;then
                success;echo 
                echo -n "Waiting for $PROG to shutdown .."
        	while checkpid $PID;do
                	echo -n "."
               		sleep 1;
                done
                success;echo
        else 
                failure;echo
        fi
    else
        echo -n $"service is dead and $PID_FILE exists.";failure;echo
        RETVAL=7
    fi    
    return $RETVAL
}
restart() {
    stop
    start
}
rhstatus() {
    status -p ${PID_FILE} $PROG
}
hid_status() {
    rhstatus >/dev/null 2>&1
}
case "$1" in
    start)
        hid_status && exit 0
        start
        ;;
    stop)
        rhstatus || exit 0
        stop
        ;;
    restart)
        restart
        ;;
    status)
        rhstatus
        RETVAL=$?
        ;;
    *)
        echo $"Usage: $0 {start|stop|status|restart}"
        RETVAL=1
esac
exit $RETVAL
