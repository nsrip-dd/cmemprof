package cmemprof

import (
	"debug/dwarf"
	"debug/elf"
	"debug/macho"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
)

var (
	symbolsInit    sync.Once
	symbols        *dwarf.Data
	populatedTable symbolTable
)

type symbolTable []tableEntry

type tableEntry struct {
	lo, hi uint64
	name   string
}

func newSymbolTable(data *dwarf.Data) symbolTable {
	var table symbolTable
	r := data.Reader()
	for {
		entry, err := r.Next()
		if err != nil || entry == nil {
			break
		}
		if entry.Tag != dwarf.TagSubprogram {
			continue
		}
		pcs, err := data.Ranges(entry)
		if err != nil {
			break
		}
		name, ok := entry.Val(dwarf.AttrName).(string)
		if !ok {
			continue
		}
		for _, pair := range pcs {
			table = append(table, tableEntry{
				lo:   pair[0],
				hi:   pair[1],
				name: name,
			})
		}
	}
	sort.Slice(table, func(i, j int) bool {
		return table[i].lo < table[j].lo
	})
	return table
}

func (s symbolTable) lookupName(pc uint64) (string, bool) {
	i := sort.Search(len(s), func(i int) bool {
		entry := s[i]
		return entry.hi >= pc
	})
	if i == len(s) {
		return "", false
	}
	if s[i].lo > pc {
		return "", false
	}
	return s[i].name, true
}

func getElfDWARF(r io.ReaderAt) (*dwarf.Data, error) {
	e, err := elf.NewFile(r)
	if err != nil {
		return nil, err
	}
	d, err := e.DWARF()
	if err != nil {
		return nil, err
	}
	return d, nil
}

func getMachoDWARF(r io.ReaderAt) (*dwarf.Data, error) {
	e, err := macho.NewFile(r)
	if err != nil {
		return nil, err
	}
	d, err := e.DWARF()
	if err != nil {
		return nil, err
	}
	return d, nil
}

func populateSymbols() {
	symbolsInit.Do(func() {
		f, err := os.Open(os.Args[0])
		if err != nil {
			fmt.Println("open executable:", err)
			return
		}
		defer f.Close()
		symbols, err = getElfDWARF(f)
		if err != nil {
			symbols, err = getMachoDWARF(f)
			if err != nil {
				return
			}
		}
		populatedTable = newSymbolTable(symbols)
	})
}

func lookupSymbol(pc uint64) (string, int, error) {
	populateSymbols()
	if symbols == nil {
		return "", 0, errors.New("no symbols")
	}
	r := symbols.Reader()
	entry, err := r.SeekPC(pc)
	if err != nil {
		return "", 0, err
	}
	lr, err := symbols.LineReader(entry)
	if lr == nil || err != nil {
		return "", 0, errors.New("no line table")
	}
	var lineEntry dwarf.LineEntry
	err = lr.SeekPC(pc, &lineEntry)
	if err != nil {
		return "", 0, err
	}
	return lineEntry.File.Name, lineEntry.Line, nil
}
