package querydigest

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"unicode/utf8"
)

type SlowQueryScanner struct {
	reader      *bufio.Reader
	line        string
	currentInfo *SlowQueryInfo
	err         error
}

func NewSlowQueryScanner(r io.Reader) *SlowQueryScanner {
	return &SlowQueryScanner{
		reader: bufio.NewReaderSize(r, 1024*1024*16),
	}
}

func (s *SlowQueryScanner) SlowQueryInfo() *SlowQueryInfo {
	return s.currentInfo
}

func (s *SlowQueryScanner) Err() error {
	return s.err
}

func (s *SlowQueryScanner) Next() bool {
	if s.err != nil {
		return false
	}
	for {
		for !strings.HasPrefix(s.line, "# Time:") {
			if err := s.nextLine(); err == io.EOF {
				return false
			} else if err != nil {
				s.err = err
				return false
			}
		}
		var slowquery SlowQueryInfo

		if err := s.nextLine(); err != nil {
			s.err = err
			return false
		}

		if err := s.nextLine(); err != nil {
			s.err = err
			return false
		}

		slowquery.QueryTime = parseQueryTime(s.line)

		for {
			if err := s.nextLine(); err == io.EOF {
				return false
			} else if err != nil {
				s.err = err
				return false
			}

			var buf string

			for {
				buf += s.line
				if strings.HasSuffix(buf, ";") {
					break
				}
				if err := s.nextLine(); err != nil {
					s.err = err
					return false
				}
			}

			if parsableQueryLine(buf) {
				slowquery.RawQuery = buf
				s.currentInfo = &slowquery
				return true
			} else if strings.HasPrefix(s.line, "#") {
				break
			}
		}
	}
}

func (s *SlowQueryScanner) nextLine() error {
	l, _, err := s.reader.ReadLine()
	if err != nil {
		return err
	}
	if utf8.Valid(l) {
		s.line = string(l)
	} else {
		s.line = fmt.Sprintf("%q", l)
	}

	return nil
}

var supportedSQLs = []string{"SELECT", "INSERT", "ALTER", "WITH", "DELETE", "UPDATE"}

func parsableQueryLine(str string) bool {
	if len(str) > 8 {
		str = str[:8]
	}
	str = strings.ToUpper(str)
	for _, s := range supportedSQLs {
		if strings.HasPrefix(str, s) {
			return true
		}
	}

	return false
}

type QueryTime struct {
	QueryTime    float64
	LockTime     float64
	RowsSent     int
	RowsExamined int
}

type SlowQueryInfo struct {
	ParsedQuery string
	RawQuery    string
	QueryTime   *QueryTime
}

func parseQueryTime(str string) *QueryTime {

	queryTimes := strings.SplitN(str, " ", 12)
	// Query_time
	qt, err := strconv.ParseFloat(queryTimes[2], 64)
	if err != nil {
		log.Fatal(err)
	}
	// Lock_time
	lt, err := strconv.ParseFloat(queryTimes[5], 64)
	if err != nil {
		log.Fatal(err)
	}
	// Rows_sent
	rs, err := strconv.ParseInt(queryTimes[7], 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	// Rows_examined
	re, err := strconv.ParseInt(queryTimes[10], 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	return &QueryTime{
		QueryTime:    qt,
		LockTime:     lt,
		RowsSent:     int(rs),
		RowsExamined: int(re),
	}
}
