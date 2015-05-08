#!/bin/sh

SERVICE_NAME=gorlim_web
DAEMON={{ GOPATH }}/bin/gorlim_web
DAEMONOPTS="-github-client={{ github_client_id }} -github-secret={{ github_client_secret }} -static-dir={{ GOPATH }}/src/{{ project }}/gorlim_web/static -authorized-keys=/home/{{ git_user }}/.ssh/authorized_keys"
PIDFILE={{ gorlim_web_pid }}

if [ ! -x $DAEMON ]; then
  echo "ERROR: Can't execute $DAEMON."
  exit 1
fi

start_service() {
  echo -n " * Starting $SERVICE_NAME... "

  PID=`$DAEMON $DAEMONOPTS > /dev/null 2>&1 & echo $!`
  #echo "Saving PID" $PID " to " $PIDFILE
  if [ -z $PID ]; then
    printf "%s\n" "Fail"
  else
    echo $PID > $PIDFILE
    printf "%s\n" "Ok"
  fi
  echo "done"
}

stop_service() {
  echo -n " * Stopping $SERVICE_NAME... "
  PID=`cat $PIDFILE`
  if [ -f $PIDFILE ]; then
      kill -9 $PID
      printf "%s\n" "Ok"
      rm -f $PIDFILE
  else
      printf "%s\n" "pidfile not found"
  fi
  echo "done"
}

status_service() {
    printf "%-50s" "Checking $SERVICE_NAME..."
    if [ -f $PIDFILE ]; then
        PID=`cat $PIDFILE`
        if [ -z "`ps axf | grep ${PID} | grep -v grep`" ]; then
            printf "%s\n" "Process dead but pidfile exists"
            exit 1 
        else
            echo "Running"
        fi
    else
        printf "%s\n" "Service not running"
        exit 3 
    fi
}

case "$1" in
  status)
    status_service
    ;;
  start)
    start_service
    ;;
  stop)
    stop_service
    ;;
  restart)
    stop_service
    start_service
    ;;
  *)
    echo "Usage: service $SERVICE_NAME {start|stop|restart|status}" >&2
    exit 1   
    ;;
esac

exit 0
