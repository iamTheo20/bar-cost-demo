package decimal

import "testing"

// Expected strings were produced by Python's decimal module (default context).
func TestArithmeticMatchesPython(t *testing.T) {
	d := MustParse
	tests := []struct {
		name string
		got  Decimal
		want string
	}{
		{"exact quotient keeps ideal exponent", d("10").Quo(d("4")), "2.5"},
		{"exact quotient keeps trailing zeros", d("2.00").Quo(d("1")), "2.00"},
		{"inexact quotient has 28 digits", d("1").Quo(d("3")), "0.3333333333333333333333333333"},
		{"inexact quotient rounds half even", d("12.90").Quo(d("11.50")), "1.121739130434782608695652174"},
		{"zero dividend", d("0").Quo(d("3.0")), "0E+1"},
		{"price change", d("12.90").Sub(d("11.50")).Quo(d("11.50")), "0.1217391304347826086956521739"},
		{"one minus a ratio", New(1, 0).Sub(d("0.2366757352941176470588235294")), "0.7633242647058823529411764706"},
		{"subtraction keeps scale", d("12.90").Sub(d("11.50")), "1.40"},
		{"product scale is the sum", d("0.0571").Mul(d("349.00")), "19.927900"},
		{"sum from zero", New(0, 0).Add(d("2094.00")), "2094.00"},
		{"quantize half even down", d("0.125").Quantize(-2), "0.12"},
		{"quantize half even up", d("0.135").Quantize(-2), "0.14"},
		{"quantize pads", d("290").Quantize(-2), "290.00"},
		{"x minus x is positive zero", d("-1.5").Sub(d("-1.5")), "0.0"},
		{"negative zero parses", d("-0"), "-0"},
		{"leading point", d(".5"), "0.5"},
		{"trailing point", d("5."), "5"},
		{"tiny is scientific", d("0.0000001"), "1E-7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if s := tt.got.String(); s != tt.want {
				t.Errorf("got %s, want %s", s, tt.want)
			}
		})
	}
}

func TestPercentMatchesPythonFormat(t *testing.T) {
	tests := []struct {
		in     string
		places int
		plus   bool
		want   string
	}{
		{"0.00125", 2, false, "0.12%"},
		{"0.00135", 2, false, "0.14%"},
		{"-0.0001", 1, true, "-0.0%"},
		{"0", 1, true, "+0.0%"},
		{"0.76", 0, false, "76%"},
		{"0.705", 0, false, "70%"},
		{"0.1217391304347826086956521739", 1, true, "+12.2%"},
		{"0.7633242647058823529411764706", 1, false, "76.3%"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := MustParse(tt.in).Percent(tt.places, tt.plus); got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestParseRefusesWhatIsNotAPlainDecimal(t *testing.T) {
	for _, in := range []string{"", "-", ".", "two", "1e3", "NaN", "1.2.3", "1 000", "1,5"} {
		t.Run(in, func(t *testing.T) {
			if _, err := Parse(in); err == nil {
				t.Errorf("Parse(%q) succeeded", in)
			}
		})
	}
}

func TestCmpIsNumeric(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"300.00", "300", 0},
		{"0.752", "0.76", -1},
		{"-0.05", "0.05", -1},
		{"0.05", "0.0500", 0},
	}
	for _, tt := range tests {
		if got := MustParse(tt.a).Cmp(MustParse(tt.b)); got != tt.want {
			t.Errorf("Cmp(%s, %s) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestQuoByZeroPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("no panic")
		}
	}()
	MustParse("1").Quo(MustParse("0.00"))
}
