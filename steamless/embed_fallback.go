//go:build !((linux && (amd64 || arm64)) || (darwin && arm64) || (windows && (amd64 || 386)))

package steamless

var embeddedBinary []byte

const embeddedBinaryName = "steamless"
