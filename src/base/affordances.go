package base

import (
	"fmt"
	"strconv"
	"regexp"
	"bytes"
	"sort"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

type byFnName []*CXFunction
type byTypName []string
type byModName []*CXModule
type byDefName []*CXDefinition
type byStrctName []*CXStruct
type byFldName []*CXField
type byParamName []*CXParameter

/*
  Lens
*/

func (s byFnName) Len() int {
    return len(s)
}
func (s byTypName) Len() int {
    return len(s)
}
func (s byModName) Len() int {
    return len(s)
}
func (s byDefName) Len() int {
    return len(s)
}
func (s byStrctName) Len() int {
    return len(s)
}
func (s byFldName) Len() int {
    return len(s)
}
func (s byParamName) Len() int {
    return len(s)
}

/*
  Swaps
*/

func (s byFnName) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}
func (s byTypName) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}
func (s byModName) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}
func (s byDefName) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}
func (s byStrctName) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}
func (s byFldName) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}
func (s byParamName) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}

/*
  Lesses
*/

func (s byFnName) Less(i, j int) bool {
    return concat(s[i].Module.Name, ".", s[i].Name) < concat(s[j].Module.Name, ".", s[j].Name)
}
func (s byTypName) Less(i, j int) bool {
    return s[i] < s[j]
}
func (s byModName) Less(i, j int) bool {
    return s[i].Name < s[j].Name
}
func (s byDefName) Less(i, j int) bool {
    return concat(s[i].Module.Name, ".", s[i].Name) < concat(s[j].Module.Name, ".", s[j].Name)
}
func (s byStrctName) Less(i, j int) bool {
    return concat(s[i].Module.Name, ".", s[i].Name) < concat(s[j].Module.Name, ".", s[j].Name)
}
func (s byFldName) Less(i, j int) bool {
    return s[i].Name < s[j].Name
}
func (s byParamName) Less(i, j int) bool {
    return s[i].Name < s[j].Name
}

func PrintAffordances (affs []*CXAffordance) {
	for i, aff := range affs {
		fmt.Printf("%d.-%s\n", i, aff.Description)
	}
}

func (aff *CXAffordance) ApplyAffordance () {
	aff.Action()
}

func FilterAffordances(affs []*CXAffordance, filters ...string) []*CXAffordance {
	filteredAffs := make([]*CXAffordance, 0)
	for _, filter := range filters {
		//re := regexp.MustCompile(regexp.QuoteMeta(filter))
		re := regexp.MustCompile("(?i)" + filter)
		for _, aff := range affs {
			if re.FindString(aff.Description) != "" {
				filteredAffs = append(filteredAffs, aff)
			}
		}
		affs = filteredAffs
		filteredAffs = make([]*CXAffordance, 0)
	}
	return affs
}

func (strct *CXStruct) GetAffordances() []*CXAffordance {
	affs := make([]*CXAffordance, 0)
	mod := strct.Module

	types := make([]string, len(BASIC_TYPES))
	copy(types, BASIC_TYPES)
	
	for _, s := range mod.Structs {
		types = append(types, s.Name)
	}

	// Getting types from imported modules
	for _, imp := range mod.Imports {
		for _, strct := range imp.Structs {
			types = append(types, concat(imp.Name, ".", strct.Name))
	       	}
	}

	// definitions for each available type
	for _, typ := range types {
		fldGensym := MakeGenSym("fld")
		fldType := typ
		
		affs = append(affs, &CXAffordance{
			Description: concat("AddField ", fldGensym, " ", typ),
			Action: func() {
				strct.AddField(MakeField(fldGensym, fldType))
			}})
	}

	return affs
}

func (expr *CXExpression) GetAffordances(settings []string) []*CXAffordance {
	// by type
	var focusNonArrays bool
	var focusArrays bool
	var focusStructs bool
	// by scope
	var focusLocals bool
	var focusGlobals bool
	
	var focusAllTypes bool
	var focusAllScopes bool
	var focusAll bool

	//extracting settings
	if len(settings) > 0 {
		for _, setting := range settings {
			if setting == "nonArrays" {focusNonArrays = true}
			if setting == "arrays" {focusArrays = true}
			if setting == "structs" {focusStructs = true}
			if setting == "locals" {focusLocals = true}
			if setting == "globals" {focusGlobals = true}
			if setting == "allScopes" {focusAllScopes = true}
			if setting == "allTypes" {focusAllTypes = true}
		}
	} else {
		focusAll = true
	}
	if (focusNonArrays && focusArrays) ||
		(!focusNonArrays && !focusArrays) {
		focusAllTypes = true
	}
	if (focusGlobals && focusLocals) ||
		(!focusGlobals && !focusLocals) {
		focusAllScopes = true
	}
	
	op := expr.Operator
	affs := make([]*CXAffordance, 0)

	// The operator for this function doesn't require arguments
	if len(op.Inputs) > 0 && len(expr.Arguments) < len(op.Inputs) {
		fn := expr.Function
		mod := expr.Module
		reqType := op.Inputs[len(expr.Arguments)].Typ // Required type for the current op's input
		defsTypes := make([]string, 0)
		args := make([]*CXArgument, 0)
		identType := "ident"

		inOutNames := make([]string, len(fn.Inputs) + 1)
		
		// Adding inputs and outputs as definitions
		if focusAll || focusAllScopes || focusLocals {
			for i, param := range fn.Inputs {
				if reqType == param.Typ || reqType == param.Typ[2:] {
					if focusAll || focusAllTypes ||
						(focusArrays && IsArray(param.Typ)) ||
						(focusNonArrays && !IsArray(param.Typ)) {
						
						inOutNames[i] = param.Name
						defsTypes = append(defsTypes, param.Typ)
						identName := encoder.Serialize(param.Name)
						args = append(args, &CXArgument{
							Typ: identType,
							Value: &identName,
						})
					}
				}
			}
		}
		
		// Adding definitions (global vars)
		if focusAll || focusAllScopes || focusGlobals {
			for _, def := range mod.Definitions {
				if reqType == def.Typ || reqType == def.Typ[2:] {
					if focusAll || focusAllTypes ||
						(focusArrays && IsArray(def.Typ)) ||
						(focusNonArrays && !IsArray(def.Typ)) {
						// we could have a var with the same name and type in global and local
						// contexts. We only want to show 1 affordance for this name
						notDuplicated := true
						for _, name := range inOutNames {
							if name == def.Name {
								notDuplicated = false
								break
							}
						}
						
						if notDuplicated {
							defsTypes = append(defsTypes, def.Typ)
							identName := encoder.Serialize(def.Name)
							args = append(args, &CXArgument{
								Typ: identType,
								Value: &identName,
							})
						}
					}
				}

				if focusAll || (focusStructs && IsStructInstance(def.Typ, expr.Module)) {
					if strct, err := expr.Context.GetStruct(def.Typ, expr.Module.Name); err == nil {
						for _, fld := range strct.Fields {
							if fld.Typ == reqType || fld.Typ[2:] == reqType {
								if focusAll || focusAllTypes ||
									(focusArrays && IsArray(fld.Typ)) ||
									(focusNonArrays && !IsArray(fld.Typ)) {
									defsTypes = append(defsTypes, fld.Typ)
									identName := encoder.Serialize(fmt.Sprintf("%s.%s", def.Name, fld.Name))

									args = append(args, &CXArgument{
										Typ: identType,
										Value: &identName,
									})
								}
							}
						}
					}
				}
					
				// if !isBasicType(def.Typ) {
				// 	if strct, err := expr.Context.GetStruct(def.Typ, expr.Module.Name); err == nil {
				// 		for _, fld := range strct.Fields {
				// 			if fld.Typ == reqType || fld.Typ[2:] == reqType {
				// 				defsTypes = append(defsTypes, fld.Typ)
				// 				identName := encoder.Serialize(fmt.Sprintf("%s.%s", def.Name, fld.Name))
								
				// 				args = append(args, &CXArgument{
				// 					Typ: identType,
				// 					Value: &identName,
				// 				})
				// 			}
				// 		}
				// 	}
				// }
			}
		}
		

		// Adding possible struct instances
		var customTypes []string
		for _, inp := range expr.Operator.Inputs {
			if !IsBasicType(inp.Typ) {
				customTypes = append(customTypes, inp.Typ)
			}
		}
		
		// Adding local definitions
		for _, ex := range expr.Function.Expressions {
			if ex == expr {
				break
			}

			// checking if it's a nonAssign local
			isNonAssign := false
			for _, outName := range ex.OutputNames {
				if len(outName.Name) > len(NON_ASSIGN_PREFIX) && outName.Name[:len(NON_ASSIGN_PREFIX)] == NON_ASSIGN_PREFIX {
					isNonAssign = true
					break
				}
			}
			if isNonAssign {
				continue
			}

			if len(ex.Operator.Outputs) != len(ex.OutputNames) ||
				len(ex.Operator.Inputs) != len(ex.Arguments) {
				// Then it's not a completed expression
				continue
			}

			if ex.Operator.Name == "initDef" {
				var typ string
				encoder.DeserializeRaw(*ex.Arguments[0].Value, &typ)

				if reqType != typ && typ[2:] != reqType {
					continue
				}
				
				val := encoder.Serialize(ex.OutputNames[0].Name)

				defsTypes = append(defsTypes, typ)
				
				args = append(args, &CXArgument{
					Typ: identType,
					Value: &val,
				})
				continue
			}

			if focusAll || focusAllScopes || focusLocals {
				for _, outName := range ex.OutputNames {
					typ := outName.Typ
					if ex.Operator.Name == "identity" {
						for _, expr := range expr.Function.Expressions {
							var identName string
							encoder.DeserializeRaw(*ex.Arguments[0].Value, &identName)
							if expr.OutputNames != nil && expr.OutputNames[0].Name == identName {
								typ = expr.OutputNames[0].Typ
								break
							}
						}
					}

					// Adding arrays of the same type
					// We'll add each slice when constructing the affordance
					if focusAll || focusAllTypes ||
						(focusArrays && IsArray(typ)) {
						if len(typ) > 2 && reqType == typ[2:] {
							defsTypes = append(defsTypes, typ)
							identName := encoder.Serialize(outName.Name)
							args = append(args, &CXArgument{
								Typ: identType,
								Value: &identName,
							})
						}
					}

					if focusAll || (focusStructs && IsStructInstance(typ, expr.Module)) {
						if strct, err := expr.Context.GetStruct(typ, expr.Module.Name); err == nil {
							for _, fld := range strct.Fields {
								if fld.Typ == reqType || fld.Typ[2:] == reqType {
									if focusAll || focusAllTypes ||
										(focusArrays && IsArray(fld.Typ)) ||
										(focusNonArrays && !IsArray(fld.Typ)) {
										defsTypes = append(defsTypes, fld.Typ)
										identName := encoder.Serialize(fmt.Sprintf("%s.%s", outName.Name, fld.Name))

										args = append(args, &CXArgument{
											Typ: identType,
											Value: &identName,
										})
									}
								}
							}
						}
					}
					
					// if !isBasicType(typ) {
					// 	if strct, err := expr.Context.GetStruct(typ, expr.Module.Name); err == nil {
					// 		for _, fld := range strct.Fields {
					// 			if fld.Typ == reqType || fld.Typ[2:] == reqType {
					// 				defsTypes = append(defsTypes, fld.Typ)
					// 				identName := encoder.Serialize(fmt.Sprintf("%s.%s", outName.Name, fld.Name))

					// 				args = append(args, &CXArgument{
					// 					Typ: identType,
					// 					Value: &identName,
					// 				})
					// 			}
					// 		}
					// 	}
					// }
					
					if reqType == typ {
						if focusAll || focusAllTypes ||
							(focusArrays && IsArray(typ)) ||
							(focusNonArrays && !IsArray(typ)) {
							defsTypes = append(defsTypes, typ)
							identName := encoder.Serialize(outName.Name)
							args = append(args, &CXArgument{
								Typ: identType,
								Value: &identName,
							})
						}
					}
				}
			}
		}

		for i, arg := range args {

			theArg := arg
			var argName string
			encoder.DeserializeRaw(*arg.Value, &argName)

			if len(defsTypes[i]) > 2 && defsTypes[i][:2] == "[]" {
				if arr, err := resolveIdent(argName, expr.Context.CallStack.Calls[len(expr.Context.CallStack.Calls) - 1]); err == nil {
					var size int32
					encoder.DeserializeAtomic((*arr.Value)[:4], &size)
					
					for c := int32(0); c < size; c++ {
						affs = append(affs, &CXAffordance{
							Description: concat("AddArgument ", argName, " ", defsTypes[i]),
							Operator: "AddArgument",
							Name: argName,
							Index: fmt.Sprintf("%d", c),
							Typ: defsTypes[i],
							Action: func() {
								expr.AddArgument(theArg)
							}})
					}
				}
				continue
			}
			
			affs = append(affs, &CXAffordance{
				Description: concat("AddArgument ", argName, " ", defsTypes[i]),
				Operator: "AddArgument",
				Name: argName,
				Typ: defsTypes[i],
				Action: func() {
					expr.AddArgument(theArg)
				}})
		}
	}

	// Output names affordances
	if len(expr.OutputNames) < len(expr.Operator.Outputs) {
		outName := MakeGenSym("var")
		affs = append(affs, &CXAffordance{
			Description: concat("AddOutputName ", outName),
			
			Operator: "AddOutputName",
			Name: outName,
			Action: func() {
				expr.AddOutputName(outName)
			}})
	}

	return affs
}

func (fn *CXFunction) GetAffordances() []*CXAffordance {
	affs := make([]*CXAffordance, 0)

	if _, ok := NATIVE_FUNCTIONS[fn.Name]; ok {
		return affs
	}
	
	mod := fn.Module
	opsNames := make([]string, 0)
	ops := make([]*CXFunction, 0)

	types := make([]string, len(BASIC_TYPES))
	copy(types, BASIC_TYPES)
	for _, s := range mod.Structs {
		types = append(types, s.Name)
	}

	// Getting types from imported modules
	for _, imp := range mod.Imports {
		for _, strct := range imp.Structs {
			types = append(types, concat(imp.Name, ".", strct.Name))
		}
	}

	// Getting operators from current module
	for _, op := range mod.Functions {
		if fn.Name != op.Name && op.Name != "main" {
			ops = append(ops, op)
			opsNames = append(opsNames, op.Name)
		}
	}

	// Getting operators from core module
	if core, err := fn.Context.GetModule(CORE_MODULE); err == nil {
		for _, op := range core.Functions {
			ops = append(ops, op)
			opsNames = append(opsNames, concat(core.Name, ".", op.Name))
		}
	}
	

	// Getting operators from imported modules
	for _, imp := range mod.Imports {
		for _, op := range imp.Functions {
			ops = append(ops, op)
			opsNames = append(opsNames, concat(imp.Name, ".", op.Name))
		}
	}

	sort.Strings(types)

	// Inputs
	for _, typ := range types {
		theTyp := typ
		theName := MakeGenSym("in")
		affs = append(affs, &CXAffordance{
			Description: concat("AddInput ", theTyp),
			Operator: "AddInput",
			Name: theName,
			Typ: theTyp,
			Action: func() {
				fn.AddInput(MakeParameter(theName, theTyp))
			}})
	}
	
	// Outputs
	for _, typ := range types {
		theTyp := typ
		theName := MakeGenSym("out")
		affs = append(affs, &CXAffordance{
			Description: concat("AddOutput ", theTyp),
			Operator: "AddOutput",
			Name: theName,
			Typ: theTyp,
			Action: func() {
				fn.AddOutput(MakeParameter(theName, theTyp))
			}})
	}

	sort.Strings(opsNames)
	sort.Sort(byFnName(ops))

	// Expressions
	for i, op := range ops {
		theOp := op
		
		var inps bytes.Buffer
		for j, inp := range ops[i].Inputs {
			if j == len(ops[i].Inputs) - 1 {
				inps.WriteString(concat(inp.Typ))
			} else {
				inps.WriteString(concat(inp.Typ, ", "))
			}
		}

		var outs bytes.Buffer
		for j, out := range ops[i].Outputs {
			if j == len(ops[i].Outputs) - 1 {
				outs.WriteString(concat(out.Typ))
			} else {
				outs.WriteString(concat(out.Typ, ", "))
			}
		}

		affs = append(affs, &CXAffordance{
			
			Description: fmt.Sprintf("AddExpression %s (%s) (%s)", opsNames[i], inps.String(), outs.String()),
			Operator: "AddExpression",
			Action: func() {
				fn.AddExpression(MakeExpression(theOp))
			}})
	}

	return affs
}

// func (fn *CXFunction) GetAffordances() []*CXAffordance {
// 	affs := make([]*CXAffordance, 0)

// if _, ok := NATIVE_FUNCTIONS[fn.Name]; ok {
// 		return affs
// 	}
	
// 	mod := fn.Module
// 	opsNames := make([]string, 0)
// 	ops := make([]*CXFunction, 0)
// 	//defs := make([]*CXDefinition, 0)
// 	// we only need the names and all of them will be of type ident
// 	defs := make([]string, 0)
// 	defsTypes := make([]*CXType, 0)

// 	types := make([]string, len(BASIC_TYPES))
// 	copy(types, BASIC_TYPES)
// 	for name, _ := range mod.Structs {
// 		types = append(types, name)
// 	}

// 	// Getting types from imported modules
// 	for impName, imp := range mod.Imports {
// 		for _, strct := range imp.Structs {
// 			types = append(types, concat(impName, ".", strct.Name))
// 		}
// 	}

// 	// Getting operators from current module
// 	for opName, op := range mod.Functions {
// 		if fn.Name != opName && opName != "main" {
// 			ops = append(ops, op)
// 			opsNames = append(opsNames, opName)
// 		}
// 	}

// 	// Getting operators from imported modules
// 	for impName, imp := range mod.Imports {
// 		for opName, op := range imp.Functions {
// 			ops = append(ops, op)
// 			opsNames = append(opsNames, concat(impName, ".", opName))
// 		}
// 	}

// 	//Getting global definitions from current module
// 	for defName, def := range mod.Definitions {
// 		defs = append(defs, defName)
// 		defsTypes = append(defsTypes, def.Typ)
// 	}

// 	//Getting global definitions from imported modules
// 	for _, imp := range mod.Imports {
// 		for defName, def := range imp.Definitions {
// 			defs = append(defs, defName)
// 			defsTypes = append(defsTypes, def.Typ)
// 		}
// 	}

// 	// Getting input defs
// 	// We might need to create an empty definition?
// 	onlyLocals := make([]string, 0)
// 	onlyLocalsTypes := make([]string, 0)
// 	for _, inp := range fn.Inputs {
// 		defs = append(defs, inp.Name)
// 		onlyLocals = append(onlyLocals, inp.Name)
// 		onlyLocalsTypes = append(onlyLocalsTypes, inp.Typ)
// 		defsTypes = append(defsTypes, inp.Typ)
// 	}

// 	// Getting output def
// 	// *why commenting it* The output definition CAN be an argument to another expr
// 	// But it SHOULD NOT be used as an argument
// 	for _, inp := range fn.Outputs {
// 		//defs = append(defs, inp.Name)
// 		onlyLocals = append(onlyLocals, inp.Name)
// 		onlyLocalsTypes = append(onlyLocalsTypes, inp.Typ)
// 		//defsTypes = append(defsTypes, inp.Typ)
// 	}

// 	// Getting local definitions
// 	for _, expr := range fn.Expressions {
		


// 		for i, outName := range expr.OutputNames {
// 			cont := true
// 			for _, def := range defs {
// 				if outName == def {
// 					cont = false
// 				}
// 			}
// 			for _, out := range fn.Outputs {
// 				if outName == out.Name {
// 					cont = false
// 				}
// 			}

// 			if cont {
// 				defs = append(defs, outName)
// 				defsTypes = append(defsTypes, expr.Operator.Outputs[i].Typ)
// 				onlyLocals = append(onlyLocals, outName)
// 				onlyLocalsTypes = append(onlyLocalsTypes, expr.Operator.Outputs[i].Typ)
// 			}
// 		}
// 	}

// 	// Input affs
// 	for _, typ := range types {
// 		affs = append(affs, &CXAffordance{
// 			Description: concat("AddInput ", typ),
// 			Action: func() {
// 				fn.AddInput(MakeParameter(MakeGenSym("in"), typ))
// 			}})
// 	}

// 	// Output affs
// 	for _, typ := range types {
// 		affs = append(affs, &CXAffordance{
// 			Description: concat("AddOutput ", typ),
// 			Action: func() {
// 				fn.AddInput(MakeParameter(MakeGenSym("out"), typ))
// 			}})
// 	}

// 	ident := "ident"
// 	for opIndex, op := range ops {
// 		theOp := op // or will keep reference to last op

// 		inputArgs := make([][]*CXArgument, 0)
// 		inputArgsTypes := make([][]string, 0)
// 		for _, inp := range theOp.Inputs {
// 			args := make([]*CXArgument, 0)
// 			argsTypes := make([]string, 0)
// 			for j, def := range defs {
// 				if defsTypes[j].Name == inp.Typ {
// 					arg := MakeArgument(MakeValue(def), ident)
// 					//arg := MakeArgument(MakeValue(def), inp.Typ)
// 					args = append(args, arg)
// 					argsTypes = append(argsTypes, inp.Typ)
// 				}
// 			}
// 			if len(args) > 0 {
// 				inputArgs = append(inputArgs, args)
// 				inputArgsTypes = append(inputArgsTypes, argsTypes)
// 			}
// 		}

// 		numberCombinations := 1
// 		for _, args := range inputArgs {
// 			numberCombinations = numberCombinations * len(args)
// 		}

// 		finalArguments := make([][]*CXArgument, numberCombinations)
// 		finalArgumentsTypes := make([][]string, numberCombinations)
// 		for i, args := range inputArgs {
// 			for j := 0; j < numberCombinations; j++ {
// 				x := 1
// 				for _, a := range inputArgs[i+1:] {
// 					x = x * len(a)
// 				}
// 				finalArguments[j] = append(finalArguments[j], args[(j / x) % len(args)])
// 				finalArgumentsTypes[j] = append(finalArgumentsTypes[j], inputArgsTypes[i][(j / x) % len(inputArgsTypes[i])])
// 			}
// 		}

// 		onlyLocals = append(onlyLocals, MakeGenSym("var"))
// 		onlyLocalsTypes = append(onlyLocalsTypes, "ident")
// 		//onlyLocals = removeDuplicates(onlyLocals)

// 		for _, args := range finalArguments {
// 			for i, local := range onlyLocals {
// 				// if a var was initialized of one type, we can't assign another type to this var later on
// 				if (onlyLocalsTypes[i] != theOp.Output.Typ &&
// 					onlyLocalsTypes[i] != "ident") &&
// 					local != fn.Output.Name {
// 					continue
// 				}
				
// 				for _, out := range theOp.Outputs {
// 					if onlyLocalsTypes
// 				}


// 				// skip affordances where the operator's output type doesn't match function's output type
// 				// and we're assigning this to the function's output var
// 				if local == fn.Output.Name && theOp.Output.Typ != fn.Output.Typ {
// 					continue
// 				}
				
// 				varExpr := local

// 				identNames := ""
// 				//fmt.Println(args)
// 				for i, arg := range args {
// 					if i == len(args) - 1 {
// 						identNames = concat(identNames, string(*arg.Value))
// 					} else {
// 						identNames = concat(identNames, string(*arg.Value), ", ")
// 					}
					
// 				}

// 				argsCopy := make([]*CXArgument, len(args))
// 				for i, arg := range args {
// 					argsCopy[i] = MakeArgumentCopy(arg)
// 					//fmt.Println(string(*argsCopy[i].Value))
// 				}

// 				affs = append(affs, &CXAffordance{
// 					Description: fmt.Sprintf("AddExpression %s = %s(%s)", varExpr, opsNames[opIndex], identNames),
// 					Action: func() {
// 						expr := MakeExpression(varExpr, theOp)
// 						fn.AddExpression(expr)
// 						for _, arg := range argsCopy {
// 							expr.AddArgument(arg)
// 						}
// 					}})
// 			}
// 		}
// 	}
	
// 	return affs
// }

func (mod *CXModule) GetAffordances() []*CXAffordance {
	affs := make([]*CXAffordance, 0)
	types := make([]string, len(BASIC_TYPES))
	copy(types, BASIC_TYPES)

	if len(mod.Structs) > 0 {
		for _, s := range mod.Structs {
			types = append(types, s.Name)
		}
	}

	// Getting types from imported modules
	for _, imp := range mod.Imports {
		for _, strct := range imp.Structs {
			types = append(types, concat(imp.Name, ".", strct.Name))
		}
	}

	// definitions for each available type
	for _, typ := range types {
		defGensym := MakeGenSym("def")
		defType := typ
		value := MakeDefaultValue(typ)
		
		affs = append(affs, &CXAffordance{
			Description: concat("AddDefinition ", defGensym, " ", typ),
			Action: func() {
				mod.AddDefinition(MakeDefinition(defGensym, value, defType))
			}})
	}

	// add imports
	for _, imp := range mod.Context.Modules {
		if imp.Name != mod.Name {
			affs = append(affs, &CXAffordance{
				Description: concat("AddImport ", imp.Name),
				Action: func() {
					mod.AddImport(imp)
				}})
		}
	}
	
	// add function
	fnGensym := MakeGenSym("fn")
	affs = append(affs, &CXAffordance{
		Description: concat("AddFunction ", fnGensym),
		Action: func() {
			mod.AddFunction(MakeFunction(fnGensym))
		}})

	// add structure
	strctGensym := MakeGenSym("strct")
	affs = append(affs, &CXAffordance{
		Description: concat("AddStruct ", strctGensym),
		Action: func() {
			mod.AddStruct(MakeStruct(strctGensym))
		}})
	
	return affs
}

func (cxt *CXProgram) GetAffordances() []*CXAffordance {
	affs := make([]*CXAffordance, 0)
	modGensym := MakeGenSym("mod")
	
	affs = append(affs, &CXAffordance {
		Description: concat("AddModule ", modGensym),
		Action: func() {
			cxt.AddModule(MakeModule(modGensym))
		}})

	// Select module
	for _, mod := range cxt.Modules {
		modName := mod.Name
		affs = append(affs, &CXAffordance {
			Description: concat("SelectModule ", modName),
			Action: func() {
				cxt.SelectModule(modName)
			}})
	}

	// Select function from current module
	if cxt.CurrentModule != nil {
		for _, fn := range cxt.CurrentModule.Functions {
			fnName := fn.Name
			affs = append(affs, &CXAffordance {
				Description: concat("SelectFunction ", fnName),
				Action: func() {
					cxt.SelectFunction(fnName)
				}})
		}
	}

	// Select struct from current module
	if cxt.CurrentModule != nil {
		for _, strct := range cxt.CurrentModule.Structs {
			strctName := strct.Name
			affs = append(affs, &CXAffordance {
				Description: concat("SelectStruct ", strctName),
				Action: func() {
					cxt.SelectStruct(strctName)
				}})
		}
	}

	// Select expression from current function
	if cxt.CurrentModule != nil && cxt.CurrentModule.CurrentFunction != nil {
		for _, expr := range cxt.CurrentModule.CurrentFunction.Expressions {
			lineNumber := expr.Line
			line := strconv.Itoa(lineNumber)
			
			affs = append(affs, &CXAffordance {
				Description: fmt.Sprintf("SelectExpression (%s.%s) Line # %s", expr.Module.Name, expr.Function.Name, line),
				Action: func() {
					cxt.SelectExpression(lineNumber)
				}})
		}
	}
	
	return affs
}
