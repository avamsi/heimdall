[[ $_github_io_avamsi_heimdall_sourced ]] && return 0
_github_io_avamsi_heimdall_sourced=true

_github_io_avamsi_heimdall_cmd='_github_io_avamsi_heimdall_nil'

_github_io_avamsi_heimdall_preexec() {
    _github_io_avamsi_heimdall_cmd=$1
    # TODO: instead of time, maybe we should get and store a unique ID?
    _github_io_avamsi_heimdall_preexec_time=$(date +%s)
    _github_io_avamsi_heimdall_preexec_id=$(
        heimdall preexec \
            --cmd="$_github_io_avamsi_heimdall_cmd" \
            --time="$_github_io_avamsi_heimdall_preexec_time"
    )
}

_github_io_avamsi_heimdall_precmd() {
    local code=$?
    [[ $_github_io_avamsi_heimdall_cmd == '_github_io_avamsi_heimdall_nil' ]] && return $code
    heimdall precmd \
        --cmd="$_github_io_avamsi_heimdall_cmd" \
        --preexec-time="$_github_io_avamsi_heimdall_preexec_time" \
        --id="$_github_io_avamsi_heimdall_preexec_id" \
        --code=$code
    # Reset back to nil since it's possible for precmd to be called without preexec (Ctrl-C, for example).
    _github_io_avamsi_heimdall_cmd='_github_io_avamsi_heimdall_nil'
    return $code
}

preexec_functions+=(_github_io_avamsi_heimdall_preexec)
precmd_functions+=(_github_io_avamsi_heimdall_precmd)
