package branding

import (
	_ "embed"
	"encoding/base64"
)

const LogoSVGPath = "/branding/codex-feishu-relay-logo.svg"

//go:embed codex_feishu_relay_logo.svg
var logoSVG []byte

var logoSVGDataURI = "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(logoSVG)

func LogoSVG() []byte {
	return logoSVG
}

func LogoSVGDataURI() string {
	return logoSVGDataURI
}
