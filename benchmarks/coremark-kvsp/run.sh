#!/usr/bin/env bash
set -euo pipefail

kvsp=${KVSP:-../../build/bin/kvsp}
out_dir=${OUT_DIR:-../../build/coremark-kvsp}
backend=${BACKEND:-iyokan}
if (( $# == 0 )); then
  set -- ruby pearl alexandrite chrysoberyl
fi

printf '%-13s %10s %10s\n' CPU CYCLES RESULT
for cpu in "$@"; do
  output="$out_dir/matrix-mul-$cpu.out"
  log="$out_dir/matrix-mul-$cpu.log"
  if ! "$kvsp" emu --backend "$backend" --cpu "$cpu" \
      --iyokan-args=--quiet "$out_dir/matrix-mul-$cpu" 5 \
      >"$output" 2>"$log"; then
    cat "$log" >&2
    exit 1
  fi
  cycles=$(awk '$1 == "#cycle" { print $2 }' "$output")
  case "$cpu" in
    ruby|pearl)
      result_register=x8
      expected=49720
      ;;
    alexandrite|chrysoberyl)
      result_register=x10
      expected=4294951480
      ;;
    *)
      printf 'unknown CPU: %s\n' "$cpu" >&2
      exit 1
      ;;
  esac
  result=$(awk -v register="$result_register" \
    '$1 == register { print $2 }' "$output")
  if [[ -z "$cycles" || "$result" != "$expected" ]]; then
    printf 'invalid %s result: cycles=%s %s=%s (expected %s)\n' \
      "$cpu" "${cycles:-missing}" "$result_register" \
      "${result:-missing}" "$expected" >&2
    exit 1
  fi
  printf '%-13s %10s %10s\n' "$cpu" "$cycles" "$result"
done
