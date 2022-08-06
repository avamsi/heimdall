```
$ go install github.com/avamsi/heimdall@latest
$ sudo $(which heimdall) bifrost install --config=$(heimdall config)
$ sudo $(which heimdall) bifrost start --config=$(heimdall config)
$ echo '# github.com/avamsi/heimdall\nsource <(heimdall sh)' >> ~/.zshrc
```
