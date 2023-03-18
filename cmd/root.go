/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"github.com/yzc1114/ChatCLI/api"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ChatCLI",
	Short: "A convenient ChatGPT CLI",
	Long:  `ChatCLI helps cli interacting with ChatGPT`,
	Args:  cobra.ArbitraryArgs,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) error { return rootRun(cmd, args) },
}

func init() {
	cobra.OnInitialize(initConfig)

	flags := rootCmd.PersistentFlags()
	flags.StringP(FlagModelAlias, "p", "GPT3.5", "Chat Model [GPT3.5]")
	flags.String(FlagOpenAiAPIKey, "", "Openai API Key [default is $OPENAPI_API_KEY$]")
	flags.BoolP(FlagInteractive, "i", false, "Interactive mode")
	_ = viper.BindPFlag(FlagOpenAiAPIKey, flags.Lookup(FlagOpenAiAPIKey))
	_ = viper.BindPFlag(FlagModelAlias, flags.Lookup(FlagModelAlias))
	_ = viper.BindPFlag(FlagInteractive, flags.Lookup(FlagInteractive))
}

func initConfig() {
	viper.AutomaticEnv()
}

const (
	FlagModelAlias   = "model"
	FlagOpenAiAPIKey = "OPENAI_API_KEY"
	FlagInteractive  = "interactive"

	Gpt35       = "GPT3.5"
	Gpt35legacy = "GPT3.5LEGACY"
	Gpt4        = "GPT4"
)

var aliasToOpenAiModel = map[string]string{
	Gpt35: "gpt-3.5-turbo",
}

var AllowedModelAlias = []string{
	Gpt35,
	//Gpt35legacy,
	//Gpt4,
}

func callAPI(modelAlias string, openaiApiKey string, message string) (string, error) {
	openaiModel := aliasToOpenAiModel[modelAlias]
	return api.ChatApi(openaiModel, openaiApiKey, message)
}

func checkModelAlias(modelAlias string) error {
	for _, m := range AllowedModelAlias {
		if m == modelAlias {
			return nil
		}
	}
	return fmt.Errorf("model %s is not supported", modelAlias)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func rootRun(cmd *cobra.Command, args []string) error {
	text := strings.Join(args, " ")
	openaiApiKey := viper.GetString(FlagOpenAiAPIKey)
	modelAlias := viper.GetString(FlagModelAlias)
	interactive := viper.GetBool(FlagInteractive)

	if len(openaiApiKey) == 0 {
		return errors.New("openai API key is not provided, set env $OPENAI_API_KEY$ or use --OPENAI_API_KEY flag")
	}
	if err := checkModelAlias(modelAlias); err != nil {
		return err
	}
	if interactive {
		return interactiveMessages(openaiApiKey, modelAlias, text)
	}

	if len(text) == 0 {
		_ = cmd.Help()
		os.Exit(0)
		return nil
	}
	return singleMessage(openaiApiKey, modelAlias, text)
}

func singleMessage(openaiApiKey string, modelAlias string, text string) error {
	response, err := callAPI(modelAlias, openaiApiKey, text)
	if err != nil {
		return err
	}
	fmt.Println(response)
	return nil
}

func interactiveMessages(openaiApiKey string, modelAlias string, firstText string) error {
	fmt.Println("Interactive mode. Ctrl+C to quit.")

	handleOneMessage := func(text string) error {
		response, err := callAPI(modelAlias, openaiApiKey, text)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	}

	if len(firstText) != 0 {
		err := handleOneMessage(firstText)
		if err != nil {
			return err
		}
	}

	for {
		templates := &promptui.PromptTemplates{
			Prompt:  "{{.}} ",
			Valid:   "",
			Invalid: "",
			Success: "",
		}

		prompt := promptui.Prompt{
			HideEntered: false,
			Label:       ">",
			Templates:   templates,
		}

		text, err := prompt.Run()

		if err != nil {
			return nil
		}
		err = handleOneMessage(text)
		if err != nil {
			return err
		}
	}
}
