package amcp

import "testing"

var formatTests = []struct {
	cmd  string
	args []interface{}
}{
	{"VERSION", []interface{}{}},
	{"THUMBNAIL", []interface{}{"LIST"}},
	{"PLAY", []interface{}{"1-1", "MY_FILE", "SLIDE", 10, "LEFT"}},
	{"PLAY", []interface{}{"demo file 124.mp3"}},
	{"DATA", []interface{}{"STORE", "key", `foo
bar`}},
	{"DATA", []interface{}{"STORE", "key", `foo\bar
foobar or "fizz"`}},
	{"MIXER", []interface{}{"1-0", "CHROMA", 1, 120, 0.1, 0, 0, 0.1, 0.1, 0.7, 0}},
}

var formatExp = []string{
	"VERSION\r\n",
	"THUMBNAIL LIST\r\n",
	"PLAY 1-1 MY_FILE SLIDE 10 LEFT\r\n",
	"PLAY \"demo file 124.mp3\"\r\n",
	"DATA STORE key \"foo\\nbar\"\r\n",
	"DATA STORE key \"foo\\\\bar\\nfoobar or \\\"fizz\\\"\"\r\n",
	"MIXER 1-0 CHROMA 1 120 0.1 0 0 0.1 0.1 0.7 0\r\n",
}

func TestFormatCmd(t *testing.T) {
	for i, test := range formatTests {
		res := formatCmd(test.cmd, test.args...)
		if res != formatExp[i] {
			t.Logf("expected: %s\ngot: %s\n", formatExp[i], res)
			t.Fail()
		}
	}
}
