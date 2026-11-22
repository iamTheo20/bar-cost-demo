// Package decimal is exact decimal money arithmetic, never float.
//
// It mirrors the subset of Python's decimal module the app relies on, under
// the default context: 28 significant digits, round half to even, signed
// zero, and Python's rules for result exponents and string forms. That is
// what makes the Go CLI print exactly what the Python version printed.
package decimal

import (
	"fmt"
	"math/big"
	"strings"
)

// Precision is the number of significant digits every operation keeps.
const Precision = 28

var (
	bigTen  = big.NewInt(10)
	bigFive = big.NewInt(5)
)

// Decimal is sign × magnitude × 10^exp. The zero value is 0.
type Decimal struct {
	neg bool
	mag *big.Int
	exp int
}

func (d Decimal) m() *big.Int {
	if d.mag == nil {
		return new(big.Int)
	}
	return d.mag
}

// New is coef × 10^exp.
func New(coef int64, exp int) Decimal {
	m := big.NewInt(coef)
	neg := coef < 0
	return Decimal{neg: neg, mag: m.Abs(m), exp: exp}
}

// Parse reads a plain decimal: optional sign, digits, optional point and
// digits, at least one digit. Exponents, NaN and Infinity are refused.
func Parse(s string) (Decimal, error) {
	raw := s
	neg := false
	if s != "" && (s[0] == '+' || s[0] == '-') {
		neg = s[0] == '-'
		s = s[1:]
	}
	intPart, frac, hasPoint := strings.Cut(s, ".")
	digits := intPart + frac
	if digits == "" || (hasPoint && strings.Contains(frac, ".")) {
		return Decimal{}, fmt.Errorf("not a decimal: %q", raw)
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return Decimal{}, fmt.Errorf("not a decimal: %q", raw)
		}
	}
	mag, _ := new(big.Int).SetString(digits, 10)
	return Decimal{neg: neg, mag: mag, exp: -len(frac)}, nil
}

// MustParse is Parse for constants; it panics on a malformed literal.
func MustParse(s string) Decimal {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

// IsZero reports whether d is zero of either sign.
func (d Decimal) IsZero() bool { return d.m().Sign() == 0 }

// Sign is -1, 0 or +1.
func (d Decimal) Sign() int {
	if d.IsZero() {
		return 0
	}
	if d.neg {
		return -1
	}
	return 1
}

// Neg is -d.
func (d Decimal) Neg() Decimal { return Decimal{neg: !d.neg, mag: d.m(), exp: d.exp} }

// Abs is |d|.
func (d Decimal) Abs() Decimal { return Decimal{mag: d.m(), exp: d.exp} }

func pow10(n int) *big.Int { return new(big.Int).Exp(bigTen, big.NewInt(int64(n)), nil) }

func numDigits(x *big.Int) int {
	if x.Sign() == 0 {
		return 1
	}
	return len(x.Text(10))
}

func (d Decimal) signed() *big.Int {
	v := new(big.Int).Set(d.m())
	if d.neg {
		v.Neg(v)
	}
	return v
}

func aligned(a, b Decimal) (*big.Int, *big.Int, int) {
	exp := min(a.exp, b.exp)
	x := new(big.Int).Mul(a.signed(), pow10(a.exp-exp))
	y := new(big.Int).Mul(b.signed(), pow10(b.exp-exp))
	return x, y, exp
}

// Cmp compares numerically: -1 when d < e, 0 when equal, +1 when d > e.
func (d Decimal) Cmp(e Decimal) int {
	x, y, _ := aligned(d, e)
	return x.Cmp(y)
}

// Add is d + e, rounded to Precision.
func (d Decimal) Add(e Decimal) Decimal {
	x, y, exp := aligned(d, e)
	sum := new(big.Int).Add(x, y)
	neg := sum.Sign() < 0
	if sum.Sign() == 0 {
		neg = d.neg && e.neg
	}
	return Decimal{neg: neg, mag: sum.Abs(sum), exp: exp}.fix()
}

// Sub is d - e, rounded to Precision.
func (d Decimal) Sub(e Decimal) Decimal { return d.Add(e.Neg()) }

// Mul is d × e, rounded to Precision.
func (d Decimal) Mul(e Decimal) Decimal {
	p := new(big.Int).Mul(d.m(), e.m())
	return Decimal{neg: d.neg != e.neg, mag: p, exp: d.exp + e.exp}.fix()
}

// Quo is d / e with Python's rules: exact quotients keep the exponent
// closest to the ideal one, others get Precision digits. It panics when e
// is zero; callers check first and say which number was zero.
func (d Decimal) Quo(e Decimal) Decimal {
	if e.IsZero() {
		panic("decimal: division by zero")
	}
	neg := d.neg != e.neg
	if d.IsZero() {
		return Decimal{neg: neg, mag: new(big.Int), exp: d.exp - e.exp}.fix()
	}
	shift := numDigits(e.m()) - numDigits(d.m()) + Precision + 1
	exp := d.exp - e.exp - shift
	var num, den *big.Int
	if shift >= 0 {
		num, den = new(big.Int).Mul(d.m(), pow10(shift)), e.m()
	} else {
		num, den = d.m(), new(big.Int).Mul(e.m(), pow10(-shift))
	}
	coef, rem := new(big.Int).QuoRem(num, den, new(big.Int))
	if rem.Sign() != 0 {
		if new(big.Int).Mod(coef, bigFive).Sign() == 0 {
			coef.Add(coef, big.NewInt(1))
		}
	} else {
		ideal := d.exp - e.exp
		r := new(big.Int)
		for exp < ideal {
			q, m := new(big.Int).QuoRem(coef, bigTen, r)
			if m.Sign() != 0 {
				break
			}
			coef = q
			exp++
		}
	}
	return Decimal{neg: neg, mag: coef, exp: exp}.fix()
}

func roundHalfEven(x *big.Int, drop int) *big.Int {
	q, r := new(big.Int).QuoRem(x, pow10(drop), new(big.Int))
	half := new(big.Int).Mul(bigFive, pow10(drop-1))
	switch c := r.Cmp(half); {
	case c > 0, c == 0 && q.Bit(0) == 1:
		q.Add(q, big.NewInt(1))
	}
	return q
}

func (d Decimal) fix() Decimal {
	n := numDigits(d.m())
	if d.IsZero() || n <= Precision {
		return d
	}
	drop := n - Precision
	mag := roundHalfEven(d.m(), drop)
	exp := d.exp + drop
	if numDigits(mag) > Precision {
		mag.Quo(mag, bigTen)
		exp++
	}
	return Decimal{neg: d.neg, mag: mag, exp: exp}
}

// Quantize rounds d half to even to exponent exp (-2 means cents).
func (d Decimal) Quantize(exp int) Decimal {
	if d.exp >= exp {
		return Decimal{neg: d.neg, mag: new(big.Int).Mul(d.m(), pow10(d.exp-exp)), exp: exp}
	}
	return Decimal{neg: d.neg, mag: roundHalfEven(d.m(), exp-d.exp), exp: exp}
}

// String is Python's str(Decimal): plain unless the exponent is positive
// or the number is very small, then scientific.
func (d Decimal) String() string {
	digits := d.m().Text(10)
	sign := ""
	if d.neg {
		sign = "-"
	}
	adjusted := d.exp + len(digits) - 1
	if d.exp <= 0 && adjusted >= -6 {
		if d.exp == 0 {
			return sign + digits
		}
		point := len(digits) + d.exp
		if point > 0 {
			return sign + digits[:point] + "." + digits[point:]
		}
		return sign + "0." + strings.Repeat("0", -point) + digits
	}
	mantissa := digits[:1]
	if len(digits) > 1 {
		mantissa += "." + digits[1:]
	}
	return fmt.Sprintf("%s%sE%+d", sign, mantissa, adjusted)
}

// Percent is Python's format(d, ".Nf%"): d × 100 rounded half to even to
// places decimals, with "%" appended. plus adds "+" to non-negative values.
// A negative value that rounds to zero keeps its "-", as Python does.
func (d Decimal) Percent(places int, plus bool) string {
	scaled := Decimal{neg: d.neg, mag: d.m(), exp: d.exp + 2}.Quantize(-places)
	digits := scaled.m().Text(10)
	if len(digits) <= places {
		digits = strings.Repeat("0", places-len(digits)+1) + digits
	}
	body := digits
	if places > 0 {
		body = digits[:len(digits)-places] + "." + digits[len(digits)-places:]
	}
	switch {
	case d.neg:
		body = "-" + body
	case plus:
		body = "+" + body
	}
	return body + "%"
}
