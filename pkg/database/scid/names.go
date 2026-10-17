package scid

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	NamePlayer = 0
	NameEvent  = 1
	NameSite   = 2
	NameRound  = 3
	NumNameTypes = 4

	NamebaseMagic = "Scid.sn\x00"
)

// NameBase holds the dictionary of names for players, events, sites, and rounds.
type NameBase struct {
	NamesByID   [NumNameTypes]map[uint32]string
	IDByName    [NumNameTypes]map[string]uint32
	Counts      [NumNameTypes]uint32
	MaxFreq     [NumNameTypes]uint32
	Frequencies [NumNameTypes]map[uint32]uint32
}

func NewNameBase() *NameBase {
	nb := &NameBase{}
	for i := 0; i < NumNameTypes; i++ {
		nb.NamesByID[i] = make(map[uint32]string)
		nb.IDByName[i] = make(map[string]uint32)
		nb.Frequencies[i] = make(map[uint32]uint32)
	}
	return nb
}

func (nb *NameBase) GetName(nt int, id uint32) string {
	if nt < 0 || nt >= NumNameTypes {
		return "?"
	}
	if name, ok := nb.NamesByID[nt][id]; ok {
		return name
	}
	return "?"
}

func (nb *NameBase) GetID(nt int, name string) (uint32, bool) {
	if nt < 0 || nt >= NumNameTypes {
		return 0, false
	}
	id, ok := nb.IDByName[nt][name]
	return id, ok
}

func (nb *NameBase) AddName(nt int, name string) uint32 {
	if id, exists := nb.GetID(nt, name); exists {
		nb.Frequencies[nt][id]++
		if nb.Frequencies[nt][id] > nb.MaxFreq[nt] {
			nb.MaxFreq[nt] = nb.Frequencies[nt][id]
		}
		return id
	}

	id := nb.Counts[nt]
	nb.Counts[nt]++
	nb.NamesByID[nt][id] = name
	nb.IDByName[nt][name] = id
	nb.Frequencies[nt][id] = 1
	if nb.MaxFreq[nt] == 0 {
		nb.MaxFreq[nt] = 1
	}
	return id
}

// ReadFromFile parses an existing .sn4 file into NameBase.
func (nb *NameBase) ReadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return nb.Read(bytes.NewReader(data))
}

func read3(r io.Reader) (uint32, error) {
	var b [3]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return (uint32(b[0]) << 16) | (uint32(b[1]) << 8) | uint32(b[2]), nil
}

func write3(w io.Writer, val uint32) error {
	b := [3]byte{
		byte(val >> 16),
		byte(val >> 8),
		byte(val),
	}
	_, err := w.Write(b[:])
	return err
}

func (nb *NameBase) Read(r io.Reader) error {
	var magic [8]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return err
	}
	if string(magic[:7]) != "Scid.sn" {
		return fmt.Errorf("invalid SN4 magic: %q", string(magic[:]))
	}

	var timeStamp uint32
	if err := binary.Read(r, binary.BigEndian, &timeStamp); err != nil {
		return err
	}

	for i := 0; i < NumNameTypes; i++ {
		count, err := read3(r)
		if err != nil {
			return err
		}
		nb.Counts[i] = count
	}

	for i := 0; i < NumNameTypes; i++ {
		mf, err := read3(r)
		if err != nil {
			return err
		}
		nb.MaxFreq[i] = mf
	}

	// Read entries for each name type
	for nt := 0; nt < NumNameTypes; nt++ {
		numNames := nb.Counts[nt]
		maxFreq := nb.MaxFreq[nt]
		prevName := ""

		for i := uint32(0); i < numNames; i++ {
			var id uint32
			if numNames >= 65536 {
				nid, err := read3(r)
				if err != nil {
					return err
				}
				id = nid
			} else {
				var nid uint16
				if err := binary.Read(r, binary.BigEndian, &nid); err != nil {
					return err
				}
				id = uint32(nid)
			}

			var freq uint32
			if maxFreq >= 65536 {
				f, err := read3(r)
				if err != nil {
					return err
				}
				freq = f
			} else if maxFreq >= 256 {
				var f uint16
				if err := binary.Read(r, binary.BigEndian, &f); err != nil {
					return err
				}
				freq = uint32(f)
			} else {
				var f uint8
				if err := binary.Read(r, binary.BigEndian, &f); err != nil {
					return err
				}
				freq = uint32(f)
			}

			var length uint8
			if err := binary.Read(r, binary.BigEndian, &length); err != nil {
				return err
			}

			var prefix uint8
			if i > 0 {
				if err := binary.Read(r, binary.BigEndian, &prefix); err != nil {
					return err
				}
			}

			suffixBytes := make([]byte, int(length)-int(prefix))
			if _, err := io.ReadFull(r, suffixBytes); err != nil {
				return err
			}

			prefixStr := ""
			if int(prefix) <= len(prevName) {
				prefixStr = prevName[:prefix]
			}
			name := prefixStr + string(suffixBytes)
			prevName = name

			nb.NamesByID[nt][id] = name
			nb.IDByName[nt][name] = id
			nb.Frequencies[nt][id] = freq
		}
	}

	return nil
}

// WriteToFile serializes the NameBase into .sn4 binary format.
func (nb *NameBase) WriteToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return nb.Write(f)
}

func (nb *NameBase) Write(w io.Writer) error {
	// Magic
	if _, err := w.Write([]byte(NamebaseMagic)); err != nil {
		return err
	}

	// Timestamp
	var timeStamp uint32 = 0
	if err := binary.Write(w, binary.BigEndian, timeStamp); err != nil {
		return err
	}

	// Counts
	for i := 0; i < NumNameTypes; i++ {
		if err := write3(w, nb.Counts[i]); err != nil {
			return err
		}
	}

	// Max frequencies
	for i := 0; i < NumNameTypes; i++ {
		if err := write3(w, nb.MaxFreq[i]); err != nil {
			return err
		}
	}

	// Write entries for each name type
	for nt := 0; nt < NumNameTypes; nt++ {
		numNames := nb.Counts[nt]
		maxFreq := nb.MaxFreq[nt]

		for id := uint32(0); id < numNames; id++ {
			// Write ID
			if numNames >= 65536 {
				if err := write3(w, id); err != nil {
					return err
				}
			} else {
				if err := binary.Write(w, binary.BigEndian, uint16(id)); err != nil {
					return err
				}
			}

			// Write freq
			freq := nb.Frequencies[nt][id]
			if maxFreq >= 65536 {
				if err := write3(w, freq); err != nil {
					return err
				}
			} else if maxFreq >= 256 {
				if err := binary.Write(w, binary.BigEndian, uint16(freq)); err != nil {
					return err
				}
			} else {
				if err := binary.Write(w, binary.BigEndian, uint8(freq)); err != nil {
					return err
				}
			}

			name := nb.NamesByID[nt][id]
			if len(name) > 255 {
				name = name[:255]
			}
			length := uint8(len(name))
			if err := binary.Write(w, binary.BigEndian, length); err != nil {
				return err
			}

			// Prefix is 0 for simple uncompressed stream
			if id > 0 {
				if err := binary.Write(w, binary.BigEndian, uint8(0)); err != nil {
					return err
				}
			}

			if _, err := w.Write([]byte(name)); err != nil {
				return err
			}
		}
	}

	return nil
}
