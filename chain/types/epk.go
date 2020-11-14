package types

import (
	"encoding"
	"fmt"
	"math/big"
	"strings"

	"github.com/EpiK-Protocol/go-epik/build"
)

type EPK BigInt

func (f EPK) String() string {
	return f.Unitless() + " EPK"
}

func (f EPK) Unitless() string {
	r := new(big.Rat).SetFrac(f.Int, big.NewInt(int64(build.FilecoinPrecision)))
	if r.Sign() == 0 {
		return "0"
	}
	return strings.TrimRight(strings.TrimRight(r.FloatString(18), "0"), ".")
}

func (f EPK) Format(s fmt.State, ch rune) {
	switch ch {
	case 's', 'v':
		fmt.Fprint(s, f.String())
	default:
		f.Int.Format(s, ch)
	}
}

func (f EPK) MarshalText() (text []byte, err error) {
	return []byte(f.String()), nil
}

func (f EPK) UnmarshalText(text []byte) error {
	p, err := ParseEPK(string(text))
	if err != nil {
		return err
	}

	f.Int.Set(p.Int)
	return nil
}

func ParseEPK(s string) (EPK, error) {
	suffix := strings.TrimLeft(s, ".1234567890")
	s = s[:len(s)-len(suffix)]
	var attoepk bool
	if suffix != "" {
		norm := strings.ToLower(strings.TrimSpace(suffix))
		switch norm {
		case "", "tepk", "epk":
		case "attoepk", "aepk", "attotepk", "atepk":
			attoepk = true
		default:
			return EPK{}, fmt.Errorf("unrecognized suffix: %q", suffix)
		}
	}

	if len(s) > 50 {
		return EPK{}, fmt.Errorf("string length too large: %d", len(s))
	}

	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return EPK{}, fmt.Errorf("failed to parse %q as a decimal number", s)
	}

	if !attoepk {
		r = r.Mul(r, big.NewRat(int64(build.FilecoinPrecision), 1))
	}

	if !r.IsInt() {
		var pref string
		if attoepk {
			pref = "atto"
		}
		return EPK{}, fmt.Errorf("invalid %stEPK value: %q", pref, s)
	}

	return EPK{r.Num()}, nil
}

func MustParseEPK(s string) EPK {
	n, err := ParseEPK(s)
	if err != nil {
		panic(err)
	}

	return n
}

var _ encoding.TextMarshaler = (*EPK)(nil)
var _ encoding.TextUnmarshaler = (*EPK)(nil)
