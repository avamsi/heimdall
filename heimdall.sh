[[ $_github_io_avamsi_heimdall_sourced ]] && return 0
_github_io_avamsi_heimdall_sourced=true

_github_io_avamsi_heimdall_cmd='_github_io_avamsi_heimdall_none'
_github_io_avamsi_heimdall_t=0

_github_io_avamsi_heimdall_preexec() {
  _github_io_avamsi_heimdall_cmd=$1
  _github_io_avamsi_heimdall_t=$(date +%s)
}

_github_io_avamsi_heimdall_precmd() {
  local r=$?
  [[ $_github_io_avamsi_heimdall_cmd == '_github_io_avamsi_heimdall_none' ]] && return $r
  heimdall notify --cmd="$_github_io_avamsi_heimdall_cmd" --t="$_github_io_avamsi_heimdall_t"
  return $r
}

preexec_functions+=(_github_io_avamsi_heimdall_preexec)
precmd_functions+=(_github_io_avamsi_heimdall_precmd)
