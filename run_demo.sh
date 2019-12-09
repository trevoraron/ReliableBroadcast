#!/bin/sh

num_clients="$(jq '.clients | length' config.json)"
tmux new-session -d -s demo
tmux select-window -t demo:0
for i in $(seq 0 $(($num_clients - 1)));
do
  if [ "$i" != "0" ]; then
    tmux split-window -v
  fi
  tmux send-keys "./client -id $i -config config.json" Enter
done
tmux select-layout tiled
tmux -2 attach-session -t demo
