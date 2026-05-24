package internal

// Version is the plugin build version. Defaults to the sentinel "0.0.0";
// goreleaser ldflag injects the real release tag at build time per
// workflow#762 plugin-version contract.
var Version = "0.0.0"
