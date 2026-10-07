package scid

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

	"newscid/pkg/chess"
)

const (
	IndexMagic     = "Scid.si\x00"
	IndexHeaderSize = 182
	IndexEntrySize  = 47

	FlagStartPos   = 1 << 0
	FlagPromo      = 1 << 1
	FlagUnderPromo = 1 << 2
	FlagDelete     = 1 << 3
)

// IndexHeader represents the 182-byte header of a .si4 file.
type IndexHeader struct {
	Magic          [8]byte
	Version        uint16
	BaseType       uint32
	NumGames       uint32
	AutoLoad       uint32
	Description    string
	CustomFlagDesc [6]string
}

// IndexEntry represents a 47-byte game record in the .si4 index.
type IndexEntry struct {
	Offset       uint32
	Length       uint32
	Flags        uint16
	WhiteID      uint32
	BlackID      uint32
	EventID      uint32
	SiteID       uint32
	RoundID      uint32
	Result       chess.Result
	VarCounts    uint16
	EcoCode      uint16
	Date         string // YYYY.MM.DD
	EventDate    string // YYYY.MM.DD
	WhiteElo     uint16
	BlackElo     uint16
	FinalMatSig  uint32
	NumHalfMoves uint16
}

// ReadIndexHeader reads the 182-byte header from reader.
func ReadIndexHeader(r io.Reader) (*IndexHeader, error) {
	h := &IndexHeader{}
	if _, err := io.ReadFull(r, h.Magic[:]); err != nil {
		return nil, err
	}
	if string(h.Magic[:7]) != "Scid.si" {
		return nil, fmt.Errorf("invalid SI4 magic: %q", string(h.Magic[:]))
	}

	if err := binary.Read(r, binary.BigEndian, &h.Version); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &h.BaseType); err != nil {
		return nil, err
	}

	numGames, err := read3(r)
	if err != nil {
		return nil, err
	}
	h.NumGames = numGames

	autoLoad, err := read3(r)
	if err != nil {
		return nil, err
	}
	h.AutoLoad = autoLoad

	descBuf := make([]byte, 108)
	if _, err := io.ReadFull(r, descBuf); err != nil {
		return nil, err
	}
	h.Description = strings.TrimRight(string(descBuf), "\x00 ")

	for i := 0; i < 6; i++ {
		flagBuf := make([]byte, 9)
		if _, err := io.ReadFull(r, flagBuf); err != nil {
			return nil, err
		}
		h.CustomFlagDesc[i] = strings.TrimRight(string(flagBuf), "\x00 ")
	}

	return h, nil
}

// WriteIndexHeader writes the 182-byte header to writer.
func WriteIndexHeader(w io.Writer, h *IndexHeader) error {
	if _, err := w.Write([]byte(IndexMagic)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.Version); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.BaseType); err != nil {
		return err
	}
	if err := write3(w, h.NumGames); err != nil {
		return err
	}
	if err := write3(w, h.AutoLoad); err != nil {
		return err
	}

	descBuf := make([]byte, 108)
	copy(descBuf, []byte(h.Description))
	if _, err := w.Write(descBuf); err != nil {
		return err
	}

	for i := 0; i < 6; i++ {
		flagBuf := make([]byte, 9)
		copy(flagBuf, []byte(h.CustomFlagDesc[i]))
		if _, err := w.Write(flagBuf); err != nil {
			return err
		}
	}

	return nil
}

// ReadIndexEntry parses a 47-byte entry from reader.
func ReadIndexEntry(r io.Reader) (*IndexEntry, error) {
	buf := make([]byte, IndexEntrySize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}

	entry := &IndexEntry{}
	entry.Offset = binary.BigEndian.Uint32(buf[0:4])
	lenLow := binary.BigEndian.Uint16(buf[4:6])
	lenHigh := buf[6]
	entry.Length = uint32(lenLow) + (uint32(lenHigh&0x80) << 9)
	entry.Flags = binary.BigEndian.Uint16(buf[7:9])

	wbHigh := buf[9]
	wLow := binary.BigEndian.Uint16(buf[10:12])
	bLow := binary.BigEndian.Uint16(buf[12:14])
	entry.WhiteID = (uint32(wbHigh>>4) << 16) | uint32(wLow)
	entry.BlackID = (uint32(wbHigh&0x0F) << 16) | uint32(bLow)

	esrHigh := buf[14]
	eLow := binary.BigEndian.Uint16(buf[15:17])
	sLow := binary.BigEndian.Uint16(buf[17:19])
	rLow := binary.BigEndian.Uint16(buf[19:21])
	entry.EventID = (uint32(esrHigh>>5) << 16) | uint32(eLow)
	entry.SiteID = (uint32((esrHigh>>2)&7) << 16) | uint32(sLow)
	entry.RoundID = (uint32(esrHigh&3) << 16) | uint32(rLow)

	entry.VarCounts = binary.BigEndian.Uint16(buf[21:23])
	resCode := (entry.VarCounts >> 12) & 3
	switch resCode {
	case 1:
		entry.Result = chess.ResultWhiteWins
	case 2:
		entry.Result = chess.ResultBlackWins
	case 3:
		entry.Result = chess.ResultDraw
	default:
		entry.Result = chess.ResultUnderway
	}

	entry.EcoCode = binary.BigEndian.Uint16(buf[23:25])
	dates := binary.BigEndian.Uint32(buf[25:29])
	dateVal := dates & 0x000FFFFF
	year := dateVal >> 9
	month := (dateVal >> 5) & 15
	day := dateVal & 31
	entry.Date = formatDate(int(year), int(month), int(day))

	entry.WhiteElo = binary.BigEndian.Uint16(buf[29:31]) & 0x0FFF
	entry.BlackElo = binary.BigEndian.Uint16(buf[31:33]) & 0x0FFF
	entry.FinalMatSig = binary.BigEndian.Uint32(buf[33:37])
	halfMoves := uint16(buf[37])

	hp0 := buf[38]
	numMovesHigh := uint16(hp0 >> 6)
	entry.NumHalfMoves = halfMoves | (numMovesHigh << 8)

	return entry, nil
}

// WriteIndexEntry writes a 47-byte entry to writer.
func WriteIndexEntry(w io.Writer, entry *IndexEntry) error {
	buf := make([]byte, IndexEntrySize)

	binary.BigEndian.PutUint32(buf[0:4], entry.Offset)
	lenLow := uint16(entry.Length & 0xFFFF)
	lenHigh := byte((entry.Length >> 16) << 7)
	binary.BigEndian.PutUint16(buf[4:6], lenLow)
	buf[6] = lenHigh
	binary.BigEndian.PutUint16(buf[7:9], entry.Flags)

	wbHigh := byte(((entry.WhiteID >> 16) << 4) | ((entry.BlackID >> 16) & 0x0F))
	buf[9] = wbHigh
	binary.BigEndian.PutUint16(buf[10:12], uint16(entry.WhiteID&0xFFFF))
	binary.BigEndian.PutUint16(buf[12:14], uint16(entry.BlackID&0xFFFF))

	esrHigh := byte(((entry.EventID >> 16) << 5) | (((entry.SiteID >> 16) & 7) << 2) | ((entry.RoundID >> 16) & 3))
	buf[14] = esrHigh
	binary.BigEndian.PutUint16(buf[15:17], uint16(entry.EventID&0xFFFF))
	binary.BigEndian.PutUint16(buf[17:19], uint16(entry.SiteID&0xFFFF))
	binary.BigEndian.PutUint16(buf[19:21], uint16(entry.RoundID&0xFFFF))

	var resCode uint16 = 0
	switch entry.Result {
	case chess.ResultWhiteWins:
		resCode = 1
	case chess.ResultBlackWins:
		resCode = 2
	case chess.ResultDraw:
		resCode = 3
	}
	varCounts := (entry.VarCounts & 0x0FFF) | (resCode << 12)
	binary.BigEndian.PutUint16(buf[21:23], varCounts)
	binary.BigEndian.PutUint16(buf[23:25], entry.EcoCode)

	dateVal := encodeDate(entry.Date)
	binary.BigEndian.PutUint32(buf[25:29], dateVal)
	binary.BigEndian.PutUint16(buf[29:31], entry.WhiteElo&0x0FFF)
	binary.BigEndian.PutUint16(buf[31:33], entry.BlackElo&0x0FFF)
	binary.BigEndian.PutUint32(buf[33:37], entry.FinalMatSig)

	buf[37] = byte(entry.NumHalfMoves & 0xFF)
	buf[38] = byte((entry.NumHalfMoves >> 8) << 6) // High 2 bits in HomePawnData[0]

	_, err := w.Write(buf)
	return err
}

func formatDate(year, month, day int) string {
	yStr := fmt.Sprintf("%04d", year)
	if year == 0 {
		yStr = "????"
	}
	mStr := fmt.Sprintf("%02d", month)
	if month == 0 {
		mStr = "??"
	}
	dStr := fmt.Sprintf("%02d", day)
	if day == 0 {
		dStr = "??"
	}
	return fmt.Sprintf("%s.%s.%s", yStr, mStr, dStr)
}

func encodeDate(dateStr string) uint32 {
	parts := strings.Split(dateStr, ".")
	if len(parts) < 3 {
		return 0
	}
	var year, month, day int
	fmt.Sscanf(parts[0], "%d", &year)
	fmt.Sscanf(parts[1], "%d", &month)
	fmt.Sscanf(parts[2], "%d", &day)
	return uint32((year << 9) | (month << 5) | day)
}

// LoadAllEntries reads the header and all index entries from an .si4 file.
func LoadAllEntries(filename string) (*IndexHeader, []*IndexEntry, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	header, err := ReadIndexHeader(f)
	if err != nil {
		return nil, nil, err
	}

	entries := make([]*IndexEntry, header.NumGames)
	for i := uint32(0); i < header.NumGames; i++ {
		entry, err := ReadIndexEntry(f)
		if err != nil {
			return header, entries[:i], fmt.Errorf("reading entry %d: %w", i+1, err)
		}
		entries[i] = entry
	}

	return header, entries, nil
}
