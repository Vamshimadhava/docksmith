package build

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// InstructionType represents the type of Docksmithfile instruction
type InstructionType string

const (
	InstructionFROM     InstructionType = "FROM"
	InstructionCOPY     InstructionType = "COPY"
	InstructionRUN      InstructionType = "RUN"
	InstructionWORKDIR  InstructionType = "WORKDIR"
	InstructionENV      InstructionType = "ENV"
	InstructionCMD      InstructionType = "CMD"
)

// Instruction represents a single line in a Docksmithfile
type Instruction struct {
	Type InstructionType
	Args []string
	Raw  string
}

// ParseDocksmithfile parses a Docksmithfile and returns a list of instructions
func ParseDocksmithfile(path string) ([]Instruction, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open Docksmithfile: %w", err)
	}
	defer f.Close()
	
	var instructions []Instruction
	scanner := bufio.NewScanner(f)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse instruction
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		
		instrType := strings.ToUpper(parts[0])
		args := parts[1:]
		
		// Handle special cases
		switch InstructionType(instrType) {
		case InstructionFROM:
			if len(args) < 1 {
				return nil, fmt.Errorf("line %d: FROM requires an image argument", lineNum)
			}
			instructions = append(instructions, Instruction{
				Type: InstructionFROM,
				Args: args,
				Raw:  line,
			})
			
		case InstructionCOPY:
			if len(args) < 2 {
				return nil, fmt.Errorf("line %d: COPY requires source and destination", lineNum)
			}
			instructions = append(instructions, Instruction{
				Type: InstructionCOPY,
				Args: args,
				Raw:  line,
			})
			
		case InstructionRUN:
			if len(args) < 1 {
				return nil, fmt.Errorf("line %d: RUN requires a command", lineNum)
			}
			// Combine all args into a single command string
			cmd := strings.TrimSpace(strings.TrimPrefix(line, "RUN"))
			instructions = append(instructions, Instruction{
				Type: InstructionRUN,
				Args: []string{cmd},
				Raw:  line,
			})
			
		case InstructionWORKDIR:
			if len(args) < 1 {
				return nil, fmt.Errorf("line %d: WORKDIR requires a path", lineNum)
			}
			instructions = append(instructions, Instruction{
				Type: InstructionWORKDIR,
				Args: args,
				Raw:  line,
			})
			
		case InstructionENV:
			if len(args) < 1 {
				return nil, fmt.Errorf("line %d: ENV requires key=value", lineNum)
			}
			// Parse ENV key=value or key value
			envStr := strings.TrimSpace(strings.TrimPrefix(line, "ENV"))
			instructions = append(instructions, Instruction{
				Type: InstructionENV,
				Args: []string{envStr},
				Raw:  line,
			})
			
		case InstructionCMD:
			// Parse CMD as JSON array ["exec","arg"]
			cmdStr := strings.TrimSpace(strings.TrimPrefix(line, "CMD"))
			
			// Check if it's JSON format
			if strings.HasPrefix(cmdStr, "[") {
				var cmdArray []string
				if err := json.Unmarshal([]byte(cmdStr), &cmdArray); err != nil {
					return nil, fmt.Errorf("line %d: invalid CMD JSON format: %w", lineNum, err)
				}
				instructions = append(instructions, Instruction{
					Type: InstructionCMD,
					Args: cmdArray,
					Raw:  line,
				})
			} else {
				// Shell form (not standard but we'll support it)
				instructions = append(instructions, Instruction{
					Type: InstructionCMD,
					Args: []string{cmdStr},
					Raw:  line,
				})
			}
			
		default:
			return nil, fmt.Errorf("line %d: unknown instruction: %s", lineNum, instrType)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	
	// Validate: first instruction must be FROM
	if len(instructions) == 0 {
		return nil, fmt.Errorf("Docksmithfile is empty")
	}
	if instructions[0].Type != InstructionFROM {
		return nil, fmt.Errorf("first instruction must be FROM")
	}
	
	return instructions, nil
}
