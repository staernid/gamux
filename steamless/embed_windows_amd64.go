//go:build windows && amd64

package steamless

import _ "embed"

//go:embed bin/steamless-windows-x64.exe
var embeddedBinary []byte

const embeddedBinaryName = "steamless.exe"
