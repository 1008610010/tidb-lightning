package mydump_test

import (
	// "fmt"
	"context"
	"io"
	"strings"

	. "github.com/pingcap/check"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-lightning/lightning/config"
	"github.com/pingcap/tidb-lightning/lightning/mydump"
	"github.com/pingcap/tidb-lightning/lightning/worker"
	"github.com/pingcap/tidb/types"
)

var _ = Suite(&testMydumpCSVParserSuite{})

type testMydumpCSVParserSuite struct {
	ioWorkers *worker.Pool
}

func (s *testMydumpCSVParserSuite) SetUpSuite(c *C) {
	s.ioWorkers = worker.NewPool(context.Background(), 5, "test_csv")
}
func (s *testMydumpCSVParserSuite) TearDownSuite(c *C) {}

type assertPosEq struct {
	*CheckerInfo
}

var posEq = &assertPosEq{
	&CheckerInfo{Name: "posEq", Params: []string{"parser", "pos", "rowID"}},
}

func (checker *assertPosEq) Check(params []interface{}, names []string) (result bool, error string) {
	parser := params[0].(mydump.Parser)
	pos, rowID := parser.Pos()
	expectedPos := int64(params[1].(int))
	expectedRowID := int64(params[2].(int))
	return pos == expectedPos && rowID == expectedRowID, ""
}

func (s *testMydumpCSVParserSuite) TestTCPH(c *C) {
	reader := strings.NewReader(
		`1|goldenrod lavender spring chocolate lace|Manufacturer#1|Brand#13|PROMO BURNISHED COPPER|7|JUMBO PKG|901.00|ly. slyly ironi|
2|blush thistle blue yellow saddle|Manufacturer#1|Brand#13|LARGE BRUSHED BRASS|1|LG CASE|902.00|lar accounts amo|
3|spring green yellow purple cornsilk|Manufacturer#4|Brand#42|STANDARD POLISHED BRASS|21|WRAP CASE|903.00|egular deposits hag|
`)

	cfg := config.CSVConfig{
		Separator:   "|",
		Delimiter:   "",
		TrimLastSep: true,
	}

	parser := mydump.NewCSVParser(&cfg, reader, config.ReadBlockSize, s.ioWorkers)

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 1,
		Row: []types.Datum{
			types.NewStringDatum("1"),
			types.NewStringDatum("goldenrod lavender spring chocolate lace"),
			types.NewStringDatum("Manufacturer#1"),
			types.NewStringDatum("Brand#13"),
			types.NewStringDatum("PROMO BURNISHED COPPER"),
			types.NewStringDatum("7"),
			types.NewStringDatum("JUMBO PKG"),
			types.NewStringDatum("901.00"),
			types.NewStringDatum("ly. slyly ironi"),
		},
	})
	c.Assert(parser, posEq, 126, 1)

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 2,
		Row: []types.Datum{
			types.NewStringDatum("2"),
			types.NewStringDatum("blush thistle blue yellow saddle"),
			types.NewStringDatum("Manufacturer#1"),
			types.NewStringDatum("Brand#13"),
			types.NewStringDatum("LARGE BRUSHED BRASS"),
			types.NewStringDatum("1"),
			types.NewStringDatum("LG CASE"),
			types.NewStringDatum("902.00"),
			types.NewStringDatum("lar accounts amo"),
		},
	})
	c.Assert(parser, posEq, 240, 2)

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 3,
		Row: []types.Datum{
			types.NewStringDatum("3"),
			types.NewStringDatum("spring green yellow purple cornsilk"),
			types.NewStringDatum("Manufacturer#4"),
			types.NewStringDatum("Brand#42"),
			types.NewStringDatum("STANDARD POLISHED BRASS"),
			types.NewStringDatum("21"),
			types.NewStringDatum("WRAP CASE"),
			types.NewStringDatum("903.00"),
			types.NewStringDatum("egular deposits hag"),
		},
	})
	c.Assert(parser, posEq, 367, 3)

	c.Assert(errors.Cause(parser.ReadRow()), Equals, io.EOF)
}

func (s *testMydumpCSVParserSuite) TestRFC4180(c *C) {
	cfg := config.CSVConfig{
		Separator: ",",
		Delimiter: `"`,
	}

	// example 1, trailing new lines

	parser := mydump.NewCSVParser(&cfg, strings.NewReader("aaa,bbb,ccc\nzzz,yyy,xxx\n"), config.ReadBlockSize, s.ioWorkers)

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 1,
		Row: []types.Datum{
			types.NewStringDatum("aaa"),
			types.NewStringDatum("bbb"),
			types.NewStringDatum("ccc"),
		},
	})
	c.Assert(parser, posEq, 12, 1)

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 2,
		Row: []types.Datum{
			types.NewStringDatum("zzz"),
			types.NewStringDatum("yyy"),
			types.NewStringDatum("xxx"),
		},
	})
	c.Assert(parser, posEq, 24, 2)

	c.Assert(errors.Cause(parser.ReadRow()), Equals, io.EOF)

	// example 2, no trailing new lines

	parser = mydump.NewCSVParser(&cfg, strings.NewReader("aaa,bbb,ccc\nzzz,yyy,xxx"), config.ReadBlockSize, s.ioWorkers)

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 1,
		Row: []types.Datum{
			types.NewStringDatum("aaa"),
			types.NewStringDatum("bbb"),
			types.NewStringDatum("ccc"),
		},
	})
	c.Assert(parser, posEq, 12, 1)

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 2,
		Row: []types.Datum{
			types.NewStringDatum("zzz"),
			types.NewStringDatum("yyy"),
			types.NewStringDatum("xxx"),
		},
	})
	c.Assert(parser, posEq, 23, 2)

	c.Assert(errors.Cause(parser.ReadRow()), Equals, io.EOF)

	// example 5, quoted fields

	parser = mydump.NewCSVParser(&cfg, strings.NewReader(`"aaa","bbb","ccc"`+"\nzzz,yyy,xxx"), config.ReadBlockSize, s.ioWorkers)

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 1,
		Row: []types.Datum{
			types.NewStringDatum("aaa"),
			types.NewStringDatum("bbb"),
			types.NewStringDatum("ccc"),
		},
	})
	c.Assert(parser, posEq, 18, 1)

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 2,
		Row: []types.Datum{
			types.NewStringDatum("zzz"),
			types.NewStringDatum("yyy"),
			types.NewStringDatum("xxx"),
		},
	})
	c.Assert(parser, posEq, 29, 2)

	c.Assert(errors.Cause(parser.ReadRow()), Equals, io.EOF)

	// example 6, line breaks within fields

	parser = mydump.NewCSVParser(&cfg, strings.NewReader(`"aaa","b
bb","ccc"
zzz,yyy,xxx`), config.ReadBlockSize, s.ioWorkers)

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 1,
		Row: []types.Datum{
			types.NewStringDatum("aaa"),
			types.NewStringDatum("b\nbb"),
			types.NewStringDatum("ccc"),
		},
	})
	c.Assert(parser, posEq, 19, 1)

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 2,
		Row: []types.Datum{
			types.NewStringDatum("zzz"),
			types.NewStringDatum("yyy"),
			types.NewStringDatum("xxx"),
		},
	})
	c.Assert(parser, posEq, 30, 2)

	c.Assert(errors.Cause(parser.ReadRow()), Equals, io.EOF)

	// example 7, quote escaping

	parser = mydump.NewCSVParser(&cfg, strings.NewReader(`"aaa","b""bb","ccc"`), config.ReadBlockSize, s.ioWorkers)

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 1,
		Row: []types.Datum{
			types.NewStringDatum("aaa"),
			types.NewStringDatum("b\"bb"),
			types.NewStringDatum("ccc"),
		},
	})
	c.Assert(parser, posEq, 19, 1)

	c.Assert(errors.Cause(parser.ReadRow()), Equals, io.EOF)
}

func (s *testMydumpCSVParserSuite) TestMySQL(c *C) {
	cfg := config.CSVConfig{
		Separator:       ",",
		Delimiter:       `"`,
		BackslashEscape: true,
		NotNull:         false,
		Null:            `\N`,
	}

	parser := mydump.NewCSVParser(&cfg, strings.NewReader(`"\"","\\","\?"
"\
",\N,\\N`), config.ReadBlockSize, s.ioWorkers)

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 1,
		Row: []types.Datum{
			types.NewStringDatum(`"`),
			types.NewStringDatum(`\`),
			types.NewStringDatum("?"),
		},
	})
	c.Assert(parser, posEq, 15, 1)

	var nullDatum types.Datum
	nullDatum.SetNull()

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 2,
		Row: []types.Datum{
			types.NewStringDatum("\n"),
			nullDatum,
			types.NewStringDatum(`\N`),
		},
	})
	c.Assert(parser, posEq, 26, 2)

	c.Assert(errors.Cause(parser.ReadRow()), Equals, io.EOF)
}

func (s *testMydumpCSVParserSuite) TestSyntaxError(c *C) {
	cfg := config.CSVConfig{
		Separator: ",",
		Delimiter: `"`,
	}

	parser := mydump.NewCSVParser(&cfg, strings.NewReader(`"???`), config.ReadBlockSize, s.ioWorkers)

	c.Assert(parser.ReadRow(), ErrorMatches, "Syntax error")
}

func (s *testMydumpCSVParserSuite) TestTSV(c *C) {
	cfg := config.CSVConfig{
		Separator:       "\t",
		Delimiter:       "",
		BackslashEscape: false,
		NotNull:         false,
		Null:            "",
		Header:          true,
	}

	parser := mydump.NewCSVParser(&cfg, strings.NewReader(`a	b	c	d	e	f
0				foo	0000-00-00
0				foo	0000-00-00
0	abc	def	ghi	bar	1999-12-31`), config.ReadBlockSize, s.ioWorkers)

	var nullDatum types.Datum
	nullDatum.SetNull()

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 1,
		Row: []types.Datum{
			types.NewStringDatum("0"),
			nullDatum,
			nullDatum,
			nullDatum,
			types.NewStringDatum("foo"),
			types.NewStringDatum("0000-00-00"),
		},
	})
	c.Assert(parser, posEq, 32, 1)
	c.Assert(parser.Columns(), DeepEquals, []string{"a", "b", "c", "d", "e", "f"})

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 2,
		Row: []types.Datum{
			types.NewStringDatum("0"),
			nullDatum,
			nullDatum,
			nullDatum,
			types.NewStringDatum("foo"),
			types.NewStringDatum("0000-00-00"),
		},
	})
	c.Assert(parser, posEq, 52, 2)

	c.Assert(parser.ReadRow(), IsNil)
	c.Assert(parser.LastRow(), DeepEquals, mydump.Row{
		RowID: 3,
		Row: []types.Datum{
			types.NewStringDatum("0"),
			types.NewStringDatum("abc"),
			types.NewStringDatum("def"),
			types.NewStringDatum("ghi"),
			types.NewStringDatum("bar"),
			types.NewStringDatum("1999-12-31"),
		},
	})
	c.Assert(parser, posEq, 80, 3)

	c.Assert(errors.Cause(parser.ReadRow()), Equals, io.EOF)
}

func (s *testMydumpCSVParserSuite) TestEmpty(c *C) {
	cfg := config.CSVConfig{
		Separator: ",",
		Delimiter: `"`,
	}

	parser := mydump.NewCSVParser(&cfg, strings.NewReader(""), config.ReadBlockSize, s.ioWorkers)
	c.Assert(errors.Cause(parser.ReadRow()), Equals, io.EOF)

	// Try again with headers.

	cfg.Header = true

	parser = mydump.NewCSVParser(&cfg, strings.NewReader(""), config.ReadBlockSize, s.ioWorkers)
	c.Assert(errors.Cause(parser.ReadRow()), Equals, io.EOF)

	parser = mydump.NewCSVParser(&cfg, strings.NewReader("h\n"), config.ReadBlockSize, s.ioWorkers)
	c.Assert(errors.Cause(parser.ReadRow()), Equals, io.EOF)
}
