// SPDX-License-Identifier: AGPL-3.0-only

package marketdata

import "testing"

func TestTimeframeCodesRoundTrip(t *testing.T) {
	for tf := TF1m; tf <= TF1M; tf++ {
		got, ok := ParseTimeframe(tf.String())
		if !ok || got != tf {
			t.Errorf("ParseTimeframe(%q) = %v, %v, want %v", tf.String(), got, ok, tf)
		}
	}
	for _, code := range []string{"", "1M ", "1d", "60m", "Timeframe(0)"} {
		if tf, ok := ParseTimeframe(code); ok {
			t.Errorf("ParseTimeframe(%q) = %v, want no match", code, tf)
		}
	}
}
