#!/bin/bash

ts(){
    date +%T
}

cleanup(){
    rm -f ./mirrored_*
}

if [ $# -eq 0 ]
  then
    echo "No arguments supplied."
    echo " Usage: ./run-mirror-traffic.sh -n|--num-of-calls INTEGER -m|--mode [rolling|static] -f|--file FILE_TO_PLAY"
    echo " where num-of-calls sets the number of parallel magic mirror sessions and mode decides if sessions are recreated or not."
fi


APPLICATION_SERVER_PORT=8443
APPLICATION_SERVER_ADDR=$(kubectl get svc webrtc-server -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
TURN_SERVER_ADDR=$(kubectl get svc stunner-gateway-udp-gateway-svc -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
# TURN_SERVER_PORT=$(kubectl get cm stunner-config -n default -o jsonpath='{.data.STUNNER_PORT}')
TURN_SERVER_PORT=3478

POSITIONAL_ARGS=()
while [[ $# -gt 0 ]]; do
  case $1 in
    -n|--num-of-calls)
      NUM_OF_CALLS="$2"
      shift # past argument
      shift # past value
      ;;
    -m|--mode)
      MODE="$2"
      shift # past argument
      shift # past value
      ;;
    -f|--file)
      FILE="$2"
      shift
      shift
      ;;
    --default)
      DEFAULT=YES
      shift # past argument
      ;;
    -*|--*)
      echo "Unknown option $1"
      exit 1
      ;;
    *)
      POSITIONAL_ARGS+=("$1") # save positional arg
      shift # past argument
      ;;
  esac
done

set -- "${POSITIONAL_ARGS[@]}" # restore positional parameters

cleanup

MIRROR_GO_CMD="go run ../webrtc-client-magic-mirror.go --turn="turn:${TURN_SERVER_ADDR}:${TURN_SERVER_PORT}" --url="wss://${APPLICATION_SERVER_ADDR}:${APPLICATION_SERVER_PORT}/magicmirror" --debug -file="${FILE}

if [[ $MODE == "static" ]]; then
    echo " Invoked with static mode."
    CALL_ID=0
    while [ $CALL_ID -lt $NUM_OF_CALLS ];
    do
        echo "[$(ts)] Setting up call with id: $CALL_ID"
        $MIRROR_GO_CMD &> call_$CALL_ID.log &
        pids[${CALL_ID}]=$! #store the parent pid

        ((CALL_ID=CALL_ID+1))
        sleep 0.7
    done

    # wait for all pids
    for pid in ${pids[*]}; do
        wait $pid
    done
    echo "[$(ts)] All calls are done. Exit."

elif [[ $MODE == "rolling" ]]; then
    echo "[$(ts)] Invoked with rolling mode."
    CURRENT_CALLS=0

    while true;
    do
        if [ $CURRENT_CALLS -lt $NUM_OF_CALLS ]; then
           $MIRROR_GO_CMD &> call_$CURRENT_CALLS.log &
           pids[${CURRENT_CALLS}]=$!
           ((CURRENT_CALLS=CURRENT_CALLS+1))

           echo "[$(ts)] Num of calls in the system: $CURRENT_CALLS, pid: $!"
           sleep 0.7
        else
            wait -n
            # echo "[$(ts)] A subprocess has been terminated."
           ((CURRENT_CALLS=CURRENT_CALLS-1))
        fi
    done

else
    echo "Unknown mode given. Try static or rolling."
fi
