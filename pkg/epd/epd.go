package epd

import (
	"fmt"
	"strconv"
	"strings"
)

// EPDRecord represents a parsed Extended Position Description (EPD) line.
type EPDRecord struct {
	FENCore   string            `json:"fenCore"`            // Pieces, turn, castling, en-passant (4 fields)
	FullFEN   string            `json:"fullFen"`            // Full 6-field standard FEN (with 0 1 default clock)
	Opcodes   map[string]string `json:"opcodes"`            // All parsed opcode: value pairs
	BestMove  string            `json:"bestMove,omitempty"`  // "bm" opcode
	AvoidMove string            `json:"avoidMove,omitempty"` // "am" opcode
	ID        string            `json:"id,omitempty"`        // "id" opcode
	Comment   string            `json:"comment,omitempty"`   // "c0" opcode
	EvalCP    *int              `json:"evalCp,omitempty"`    // "ce" opcode (centipawns)
}

// Record is an alias for EPDRecord.
type Record = EPDRecord

// Parse parses an EPD string into a structured EPDRecord.
func Parse(epdLine string) (*EPDRecord, error) {
	line := strings.TrimSpace(epdLine)
	if line == "" {
		return nil, fmt.Errorf("empty EPD line")
	}

	parts := strings.Split(line, " ")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid EPD: expected at least 4 board fields, got %d", len(parts))
	}

	fenCore := strings.Join(parts[0:4], " ")
	fullFEN := fenCore + " 0 1"

	rec := &EPDRecord{
		FENCore: fenCore,
		FullFEN: fullFEN,
		Opcodes: make(map[string]string),
	}

	if len(parts) > 4 {
		remainder := strings.Join(parts[4:], " ")
		parseOpcodes(remainder, rec)
	}

	return rec, nil
}

type epdOpcode uint8

const (
	opOther epdOpcode = iota
	opBM
	opAM
	opID
	opC0
	opCE
)

func classifyOpcode(op string) epdOpcode {
	switch op {
	case "bm":
		return opBM
	case "am":
		return opAM
	case "id":
		return opID
	case "c0":
		return opC0
	case "ce":
		return opCE
	default:
		return opOther
	}
}

// parseOpcodes extracts opcode value pairs separated by semicolons.
func parseOpcodes(s string, rec *EPDRecord) {
	rawOps := strings.Split(s, ";")
	for _, raw := range rawOps {
		op := strings.TrimSpace(raw)
		if op == "" {
			continue
		}

		spaceIdx := strings.IndexByte(op, ' ')
		var opcode, operand string
		if spaceIdx == -1 {
			opcode = op
			operand = ""
		} else {
			opcode = op[:spaceIdx]
			operand = strings.TrimSpace(op[spaceIdx+1:])
			// Strip outer quotes if any
			if len(operand) >= 2 && operand[0] == '"' && operand[len(operand)-1] == '"' {
				operand = operand[1 : len(operand)-1]
			}
		}

		rec.Opcodes[opcode] = operand

		switch classifyOpcode(opcode) {
		case opBM:
			rec.BestMove = operand
		case opAM:
			rec.AvoidMove = operand
		case opID:
			rec.ID = operand
		case opC0:
			rec.Comment = operand
		case opCE:
			if cp, err := strconv.Atoi(operand); err == nil {
				rec.EvalCP = &cp
			}
		}
	}
}

// Format formats a Record or FEN with opcodes back into standard EPD notation.
func Format(fen string, opcodes map[string]string) string {
	parts := strings.Split(strings.TrimSpace(fen), " ")
	fenCore := fen
	if len(parts) >= 4 {
		fenCore = strings.Join(parts[:4], " ")
	}

	if len(opcodes) == 0 {
		return fenCore
	}

	var sb strings.Builder
	sb.WriteString(fenCore)

	for op, val := range opcodes {
		sb.WriteByte(' ')
		sb.WriteString(op)
		if val != "" {
			sb.WriteByte(' ')
			if strings.IndexByte(val, ' ') != -1 || classifyOpcode(op) == opID || (len(op) > 0 && op[0] == 'c') {
				sb.WriteByte('"')
				sb.WriteString(val)
				sb.WriteByte('"')
			} else {
				sb.WriteString(val)
			}
		}
		sb.WriteByte(';')
	}

	return sb.String()
}
