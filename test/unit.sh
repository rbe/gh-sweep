#!/usr/bin/env bash
# Logic tests for gh-sweep. No terminal and no GitHub calls.
# shellcheck disable=SC2034,SC2154
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/.." && pwd)
# shellcheck source=/dev/null
source "$ROOT/gh-sweep"

fail=0
ok() { printf 'ok  %s\n' "$1"; }
bad() { printf 'FAIL %s\n' "$1" >&2; fail=1; }

assert_eq() {
  local name=$1 got=$2 want=$3
  if [[ $got == "$want" ]]; then
    ok "$name"
  else
    bad "$name"
    printf '  got  [%s]\n  want [%s]\n' "$got" "$want" >&2
  fi
}

assert_ok() {
  if "$@"; then
    ok "$1"
  else
    bad "$1"
  fi
}

screen_has() {
  local needle=$1 line
  for line in "${screen_lines[@]}"; do
    if [[ $line == *"$needle"* ]]; then
      return 0
    fi
  done
  return 1
}

setup_ui() {
  ascii_flag=1
  no_color_flag=1
  auto_expand=0
  export GH_SWEEP_COLS=100
  export GH_SWEEP_ROWS=36
  export TZ=UTC
  tmpdir=""
  init_runtime
  NOW_EPOCH=$(iso_to_epoch "2026-11-14T12:00:00Z")
  parse_repo acme/widgets
}

load_fixtures() {
  ART_RAW=$(jq -r "$(artifact_jq)" "$ROOT/test/fixtures/artifacts.json")
  CACHE_RAW=$(jq -r "$(cache_jq)" "$ROOT/test/fixtures/caches.json")
  art_error=""
  cache_error=""
  first_build=1
  apply_streams
  build_view
  place_cursor "" ""
  ensure_scroll
  mode=list
  return_mode=list
  compose_screen
}

item_on_id() {
  local want=$1 i
  for ((i = 0; i < n_items; i++)); do
    if [[ ${item_id[$i]} == "$want" && ${item_kind[$i]} == "${2:-artifact}" ]]; then
      printf '%s' "${item_on[$i]}"
      return
    fi
  done
  printf missing
}

# --- formatting ----------------------------------------------------------

setup_ui
assert_eq "epoch Z" "$(iso_to_epoch 2020-01-10T14:59:22Z)" 1578668362
assert_eq "epoch offset" "$(iso_to_epoch 2020-01-10T14:59:22+02:00)" 1578661162
assert_eq "epoch fraction" "$(iso_to_epoch 2020-01-10T14:59:22.0766667Z)" 1578668362
assert_eq "size bytes" "$(human_size 0)" "0 B"
assert_eq "size kb" "$(human_size 1536)" "1.5 KB"
assert_eq "size gb" "$(human_size 1073741824)" "1.0 GB"
assert_eq "age days" "$(human_time "$(iso_to_epoch 2026-11-12T12:00:00Z)")" "2d ago"
assert_eq "ref branch" "$(shorten_ref refs/heads/main)" main
assert_eq "ref pr" "$(shorten_ref refs/pull/12/merge)" pr/12

parse_repo acme/widgets
assert_eq "repo owner" "$owner" acme
assert_eq "repo name" "$name" widgets
assert_eq "repo nwo" "$repo_nwo_full" acme/widgets
if parse_repo 'not a repo'; then
  bad "reject bad repo"
else
  ok "reject bad repo"
fi
if parse_repo github.com/acme/widgets; then
  assert_eq "github.com host stripped" "$repo_nwo_full" acme/widgets
else
  bad "github.com/owner/repo"
fi
if parse_repo ghe.example.com/acme/widgets; then
  assert_eq "enterprise host" "$api_hostname" ghe.example.com
  assert_eq "enterprise nwo" "$repo_nwo_full" ghe.example.com/acme/widgets
else
  bad "enterprise repo"
fi

# --- inventory -----------------------------------------------------------

setup_ui
load_fixtures
assert_eq "artifact count" "$(printf '%s\n' "$ART_RAW" | grep -c $'\037' || true)" 4
if screen_has "Weird Name" && ! screen_has $'\t'; then
  ok "tab stripped from name"
else
  bad "tab stripped from name"
fi
if screen_has "dist" && screen_has "coverage"; then
  ok "names visible"
else
  bad "names visible"
  printf '%s\n' "${screen_lines[@]}" >&2
fi

dist_at=0
cov_at=0
i=0
for line in "${screen_lines[@]}"; do
  if [[ $line == *dist* && $dist_at == 0 ]]; then
    dist_at=$i
  fi
  if [[ $line == *coverage* ]]; then
    cov_at=$i
  fi
  i=$((i + 1))
done
if ((dist_at > 0 && cov_at > dist_at)); then
  ok "largest artifact group first"
else
  bad "largest artifact group first"
fi

# Cursor starts on the dist group, not the select-all section.
if screen_has ">[ ]" && screen_has "dist"; then
  ok "cursor on a group"
else
  bad "cursor on a group"
fi
toggle_current
compose_screen
if screen_has ">[x]"; then
  ok "group checkbox"
else
  bad "group checkbox"
  printf '%s\n' "${screen_lines[@]}" >&2
fi
assert_eq "both dist runs checked" "$(item_on_id 101)" 1
assert_eq "older dist checked" "$(item_on_id 100)" 1
assert_eq "coverage untouched" "$(item_on_id 90)" 0

open_current
compose_screen
id101=0
id100=0
i=0
for line in "${screen_lines[@]}"; do
  if [[ $line == *"#101"* ]]; then id101=$i; fi
  if [[ $line == *"#100"* ]]; then id100=$i; fi
  i=$((i + 1))
done
if ((id101 > 0 && id100 > id101)); then
  ok "newer run listed first"
else
  bad "newer run listed first"
fi
exp_flag=missing
for ((i = 0; i < n_items; i++)); do
  if [[ ${item_id[$i]} == 90 && ${item_kind[$i]} == artifact ]]; then
    exp_flag=${item_expired[$i]}
  fi
done
assert_eq "expired flag" "$exp_flag" 1
set_all_open 1
compose_screen
if screen_has "expired"; then
  ok "expired marker"
else
  bad "expired marker"
fi

# Partial group checkbox.
toggle_item_by_id() {
  local id=$1 i
  for ((i = 0; i < n_items; i++)); do
    if [[ ${item_id[$i]} == "$id" ]]; then
      item_on[$i]=0
    fi
  done
}
toggle_item_by_id 100
compose_screen
if screen_has "[-]"; then
  ok "partial group"
else
  bad "partial group"
fi

# Keep the newest of each name: only the older dist run is selected.
select_except_newest 1
assert_eq "keep newest leaves 101" "$(item_on_id 101)" 0
assert_eq "keep newest selects 100" "$(item_on_id 100)" 1
assert_eq "keep newest leaves coverage" "$(item_on_id 90)" 0
select_expired
assert_eq "expired selected" "$(item_on_id 90)" 1

# Filter is literal, not a glob.
filter="dist[1]"
filter_l="dist[1]"
build_view
if ((${#view_type[@]} == 0)); then
  ok "filter brackets are literal"
else
  bad "filter brackets are literal"
fi
filter="release"
filter_l="release"
build_view
compose_screen
if screen_has "dist" && ! screen_has "coverage"; then
  ok "filter by branch"
else
  bad "filter by branch"
fi
filter=""
filter_l=""

# --- projects ------------------------------------------------------------

setup_ui
add_project acme/widgets
add_project other/lib
add_project acme/widgets
assert_eq "two projects" "$n_proj" 2
rebuild_proj_view
mode=projects
compose_screen
if screen_has "acme/widgets" && screen_has "other/lib" && screen_has "not scanned"; then
  ok "project list"
else
  bad "project list"
  printf '%s\n' "${screen_lines[@]}" >&2
fi
if screen_has "enter picks"; then
  ok "project hint"
else
  bad "project hint"
fi
toggle_project_at_cursor
compose_screen
if screen_has ">[x]"; then
  ok "project checkbox"
else
  bad "project checkbox"
fi

# Biggest-first order, and a full project wipe does not touch the other repo.
art_stream=$(jq -r "$(artifact_jq)" "$ROOT/test/fixtures/artifacts.json")
cache_stream=$(jq -r "$(cache_jq)" "$ROOT/test/fixtures/caches.json")
printf '%s\n' "$art_stream" > "$(proj_art_file 0)"
printf '%s\n' "$cache_stream" > "$(proj_cache_file 0)"
printf '%s\n' $'T 1\n55\037notes\037100\0372026-11-01T00:00:00Z\0370\037main\0371' > "$(proj_art_file 1)"
proj_scanned[0]=1
proj_scanned[1]=1
proj_art_n[0]=4
proj_cache_n[0]=3
proj_art_bytes[0]=1512002048
proj_cache_bytes[0]=2150000000
proj_exp_bytes[0]=12000000
proj_exp_n[0]=1
proj_on[0]=1
proj_on[1]=0
WIPE_COUNT=1
log=$tmpdir/deletes.log
: > "$log"
gh() {
  printf '%s\n' "$*" >> "$log"
  return 0
}
perform_project_wipes
mapfile -t deleted < <(sed -n 's/.*artifacts\/\([0-9]*\).*/art \1/p; s/.*cache delete .* \([0-9][0-9]*\).*/cache \1/p' "$log")
# shellcheck disable=SC2207
got=$(printf '%s\n' "${deleted[@]}")
want=$(printf '%s\n' "cache 7" "art 101" "art 100" "cache 8" "cache 9" "art 90" "art 80")
assert_eq "wipe deletes largest first" "$got" "$want"
if [[ $got == *55* ]]; then
  bad "other project untouched"
else
  ok "other project untouched"
fi
assert_eq "result mode" "$mode" result
if screen_has "6-12 hours"; then
  ok "quota lag note"
else
  compose_screen
  if screen_has "6-12 hours"; then
    ok "quota lag note"
  else
    bad "quota lag note"
  fi
fi

# Strict confirm for one large project asks for the repo name, not y.
setup_ui
add_project acme/widgets
parse_repo acme/widgets
ART_RAW=$art_stream
CACHE_RAW=$cache_stream
first_build=1
apply_streams
select_all_live
begin_confirm
assert_eq "large selection is strict" "$confirm_strict" 1
assert_eq "confirm token" "$confirm_token" widgets
KEY=y
handle_key
assert_eq "y does not delete a large selection" "$mode" confirm
confirm_buf=widgets
KEY=$'\r'
handle_key
# gh is still the function above, but a new shell function persists.
assert_eq "typed name confirms" "$mode" result

printf '\n%d failed\n' "$fail"
exit "$fail"
