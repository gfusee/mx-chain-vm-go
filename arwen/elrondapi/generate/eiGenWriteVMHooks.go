package elrondapigenerate

import (
	"fmt"
	"os"
)

func eiGroupInterfaceName(group *EIGroup) string {
	return fmt.Sprintf("%sVMHooks", group.Name)
}

func WriteEIInterface(out *os.File, eiMetadata *EIMetadata) {
	out.WriteString("package executor \n\n")
	out.WriteString("// Code generated by elrondapi generator. DO NOT EDIT.\n\n")
	out.WriteString("// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n")
	out.WriteString("// !!!!!!!!!!!!!!!!!!!!!! AUTO-GENERATED FILE !!!!!!!!!!!!!!!!!!!!!!\n")
	out.WriteString("// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n")
	out.WriteString("\n")
	out.WriteString("// VMHooks contains all VM functions that can be called by the executor during SC execution.\n")
	out.WriteString("type VMHooks interface {\n")

	for _, group := range eiMetadata.Groups {
		out.WriteString(fmt.Sprintf("\t%s\n", eiGroupInterfaceName(group)))
	}

	out.WriteString("}\n")

	for _, group := range eiMetadata.Groups {
		out.WriteString(fmt.Sprintf("\ntype %s interface {\n", eiGroupInterfaceName(group)))

		for _, funcMetadata := range group.Functions {
			out.WriteString(fmt.Sprintf("\t%s(", upperInitial(funcMetadata.Name)))
			for argIndex, arg := range funcMetadata.Arguments {
				if argIndex > 0 {
					out.WriteString(", ")
				}
				out.WriteString(fmt.Sprintf("%s %s", arg.Name, arg.Type))
			}
			out.WriteString(")")
			if funcMetadata.Result != nil {
				out.WriteString(fmt.Sprintf(" %s", funcMetadata.Result.Type))
			}

			out.WriteString("\n")
		}

		out.WriteString("}\n")
	}

}

func externResult(functResult *EIFunctionResult) string {
	if functResult == nil {
		return "void"
	}
	return cgoType(functResult.Type)
}
