[[ $_github_io_avamsi_heimdall_sourced ]] && return 0
_github_io_avamsi_heimdall_sourced=true

_github_io_avamsi_heimdall_cmd='_github_io_avamsi_heimdall_nil'
_github_io_avamsi_heimdall_start_time=0

_github_io_avamsi_heimdall_preexec() {
    _github_io_avamsi_heimdall_cmd=$1
    _github_io_avamsi_heimdall_start_time=$(date +%s)
}

_github_io_avamsi_heimdall_precmd() {
    local code=$?
    [[ $_github_io_avamsi_heimdall_cmd == '_github_io_avamsi_heimdall_nil' ]] && return $code
    heimdall notify \
        --cmd="$_github_io_avamsi_heimdall_cmd" \
        --start_time="$_github_io_avamsi_heimdall_start_time" \
        --code=$code
    # Reset back to nil since it's possible for precmd to be called without preexec (Ctrl-C, for example).
    _github_io_avamsi_heimdall_cmd='_github_io_avamsi_heimdall_nil'
    return $code
}

preexec_functions+=(_github_io_avamsi_heimdall_preexec)
precmd_functions+=(_github_io_avamsi_heimdall_precmd)
