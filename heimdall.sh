[[ $_github_io_avamsi_heimdall_sourced ]] && return 0
_github_io_avamsi_heimdall_sourced=true

_github_io_avamsi_heimdall_cmd='_github_io_avamsi_heimdall_nil'

_github_io_avamsi_heimdall_preexec() {
    _github_io_avamsi_heimdall_cmd=$1
    _github_io_avamsi_heimdall_preexec_time=$(date +%s)
    _github_io_avamsi_heimdall_id=$(
        heimdall start \
            --cmd="$_github_io_avamsi_heimdall_cmd" \
            --time="$_github_io_avamsi_heimdall_preexec_time"
    )
}

_github_io_avamsi_heimdall_precmd() {
    local code=$?
    [[ $_github_io_avamsi_heimdall_cmd == '_github_io_avamsi_heimdall_nil' ]] && return $code
    heimdall end \
        --cmd="$_github_io_avamsi_heimdall_cmd" \
        --start-time="$_github_io_avamsi_heimdall_preexec_time" \
        --code=$code \
        --id="$_github_io_avamsi_heimdall_id"
    # Reset back to nil since it's possible for precmd to be called without preexec (Ctrl-C, for example).
    _github_io_avamsi_heimdall_cmd='_github_io_avamsi_heimdall_nil'
    return $code
}

preexec_functions+=(_github_io_avamsi_heimdall_preexec)
precmd_functions+=(_github_io_avamsi_heimdall_precmd)
