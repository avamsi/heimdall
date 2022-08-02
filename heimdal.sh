[[ $_github_io_avamsi_heimdal_sourced ]] && return 0
_github_io_avamsi_heimdal_sourced=true

_github_io_avamsi_heimdal_cmd='_github_io_avamsi_heimdal_none'
_github_io_avamsi_heimdal_t=0

_github_io_avamsi_heimdal_preexec() {
  _github_io_avamsi_heimdal_cmd=$1
  _github_io_avamsi_heimdal_t=$(date +%s)
}

_github_io_avamsi_heimdal_precmd() {
  local r=$?
  [[ $_github_io_avamsi_heimdal_cmd == '_github_io_avamsi_heimdal_none' ]] && return $r
  heimdal --cmd="$_github_io_avamsi_heimdal_cmd" --t="$_github_io_avamsi_heimdal_t"
  return $r
}

preexec_functions+=(_github_io_avamsi_heimdal_preexec)
precmd_functions+=(_github_io_avamsi_heimdal_precmd)
