package cfg

/*
Uses Viper to read values from ENV variable or commandline flags transparently.

a flag with name "x-y" can be set in CLI as <binary> --x-y
if the same flag has to be set in ENV, it has tobe set as X_Y
*/

import (
	"flag"
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"strings"
)

// StructPath is path for virtualmachineconfig package
var StructPath string

func init() {
	flag.String("sdk-path", "", "path for sdk repo")

	//pflag configuration
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	viper.BindPFlags(pflag.CommandLine)
	pflag.Parse()
	pflag.Visit(func(f *pflag.Flag) {
		fmt.Printf("cfg package: %s overridden: %s -> %s\n", f.Name, f.DefValue, f.Value)
	})

	//Env configuration
	viper.AutomaticEnv()
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	//Config Init
	StructPath = viper.GetString("sdk-path")
}
