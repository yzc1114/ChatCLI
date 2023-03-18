package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/c-bata/go-prompt"
	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/gosuri/uilive"
	"github.com/yzc1114/ChatCLI/api"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	FlagModelAlias   = "OPENAI_CHAT_MODEL"
	FlagOpenAiAPIKey = "OPENAI_API_KEY"
	FlagRenderStyle  = "MD_RENDER_STYLE"
	FlagInteractive  = "interactive"
	FlagTimeout      = "timeout"

	FlagRenderPlainText = "plain-text"

	Gpt35 = "GPT3.5"
	Gpt4  = "GPT4"

	MarkdownStyleDark    = "dark"
	MarkdownStyleNoTTY   = "notty"
	MarkdownStyleLight   = "light"
	MarkdownStyleDracula = "dracula"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ChatCLI",
	Short: "A convenient ChatGPT CLI",
	Long:  `ChatCLI helps interact with ChatGPT in cli`,
	Example: `
Ask single message: ChatCLI [Content]

Enter interactive mode: ChatCLI -i [Optional: First Sentence]

Use Dracula markdown rendering style: ChatCLI -s dracula [Content]

Set timeout to 30 seconds, enter interactive mode: ChatCLI -t 30 -i
`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error { return rootRun(cmd, args) },
}

func init() {
	viper.AutomaticEnv()
	flags := rootCmd.PersistentFlags()

	flags.StringP(FlagModelAlias, "m", "GPT3.5", fmt.Sprintf("GPT model. options: %v. It can be set in env var $%s$", allowedModelAlias, FlagModelAlias))
	flags.String(FlagOpenAiAPIKey, "", fmt.Sprintf("Openai API Key. It can be set in env var $%s$", FlagOpenAiAPIKey))
	flags.BoolP(FlagRenderPlainText, "p", false, "Render GPT output as plain text. Default is markdown.")
	flags.StringP(FlagRenderStyle, "s", MarkdownStyleDark, fmt.Sprintf("Style of markdown rendering. options: %v. It can be set in env var $%s$", allowedMarkdownStyles, FlagRenderStyle))
	flags.BoolP(FlagInteractive, "i", false, "Interactive mode. Use ` to start multi-line input.")
	flags.Int64P(FlagTimeout, "t", 60, "Timeout second. Default is 60 seconds.")

	for _, f := range []string{FlagOpenAiAPIKey, FlagModelAlias, FlagRenderPlainText, FlagInteractive, FlagRenderStyle, FlagTimeout} {
		_ = viper.BindPFlag(f, flags.Lookup(f))
	}
}

var aliasToOpenAiModel = map[string]string{
	Gpt35: "gpt-3.5-turbo",
}

var allowedModelAlias = []string{
	Gpt35,
	//Gpt35legacy,
	//Gpt4,
}

var allowedMarkdownStyles = []string{
	MarkdownStyleDark,
	MarkdownStyleLight,
	MarkdownStyleDracula,
	MarkdownStyleNoTTY,
}

func callAPIDisposable(modelAlias string, openaiApiKey string, message string) (string, error) {
	return callAPI(modelAlias, openaiApiKey, []api.Msg{{
		Role:    api.User,
		Content: message,
	}})
}

func callAPI(modelAlias string, openaiApiKey string, msgs []api.Msg) (string, error) {
	openaiModel := aliasToOpenAiModel[modelAlias]
	timeoutDuration := time.Duration(viper.GetInt64(FlagTimeout)) * time.Second
	type Result struct {
		resp string
		err  error
	}
	ch := make(chan Result)
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()
	go func(ctx context.Context) {
		resp, err := api.ChatApi(openaiModel, openaiApiKey, msgs)
		ch <- Result{resp, err}
	}(ctx)

	writer := uilive.New()
	writer.Start()

	clearLine := func() {
		var ESC = 27
		var clear = fmt.Sprintf("%c[%dA%c[2K", ESC, 1, ESC)
		_, _ = fmt.Printf(clear)
	}

	defer func() {
		writer.Stop()
		clearLine()
	}()

	updateWaitingLine := func() func() {
		loadSym := []string{"-", "\\", "|", "/"}
		loadSymI := 0
		return func() {
			_, _ = color.New(color.FgCyan).Fprintf(writer, "%s Thinking... ", viper.GetString(FlagModelAlias))
			_, _ = color.New(color.FgRed).Fprintln(writer, loadSym[loadSymI%len(loadSym)])
			loadSymI += 1
		}
	}()

	for {
		select {
		case r := <-ch:
			return r.resp, r.err
		case <-ctx.Done():
			return "", fmt.Errorf("timeout in %d seconds", timeoutDuration/time.Second)
		default:
			updateWaitingLine()
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func printCallAPIError(err error) {
	_, _ = color.New(color.FgRed).Printf("error calling openai API: %s\n", err)
}

func checkModelAlias(modelAlias string) error {
	for _, m := range allowedModelAlias {
		if m == modelAlias {
			return nil
		}
	}
	return fmt.Errorf("model %s is not supported. allowed models: %v", modelAlias, allowedModelAlias)
}

func checkMDRenderStyle(renderStyle string) error {
	for _, m := range allowedMarkdownStyles {
		if m == renderStyle {
			return nil
		}
	}
	return fmt.Errorf("render style %s is not supported. allowed styles: %v", renderStyle, allowedMarkdownStyles)
}

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
	renderStyle := viper.GetString(FlagRenderStyle)
	interactive := viper.GetBool(FlagInteractive)

	if len(openaiApiKey) == 0 {
		return errors.New("openai API key is not provided, set env $OPENAI_API_KEY$ or use --OPENAI_API_KEY flag")
	}
	if err := checkModelAlias(modelAlias); err != nil {
		return err
	}
	if err := checkMDRenderStyle(renderStyle); err != nil {
		return err
	}
	if interactive {
		interactiveMessages(openaiApiKey, modelAlias, text)
		return nil
	}

	if len(text) == 0 {
		_ = cmd.Help()
		os.Exit(0)
		return nil
	}
	singleMessage(openaiApiKey, modelAlias, text)
	return nil
}

func singleMessage(openaiApiKey string, modelAlias string, text string) {
	response, err := callAPIDisposable(modelAlias, openaiApiKey, text)
	if err != nil {
		printCallAPIError(err)
	}
	render(response)
}

func interactiveMessages(openaiApiKey string, modelAlias string, firstText string) {
	fmt.Println("Interactive mode. Use ` to start multi-line input. Ctrl+C to quit.")

	records := make([]api.Msg, 0)

	handleUserMsg := func(text string) error {
		records = append(records, api.Msg{
			Role:    api.User,
			Content: text,
		})
		response, err := callAPI(modelAlias, openaiApiKey, records)
		if err != nil {
			printCallAPIError(err)
			return err
		}
		render(response)
		records = append(records, api.Msg{
			Role:    api.System,
			Content: response,
		})
		return nil
	}

	if len(firstText) != 0 {
		err := handleUserMsg(firstText)
		if err != nil {
			return
		}
	}

	var inputBuffer []string
	var isMultiline bool

	var executor = func(s string) error {
		if !isMultiline && strings.HasPrefix(s, "`") {
			// Handle multi-line input
			inputBuffer = append(inputBuffer, strings.TrimPrefix(s, "`"))
			isMultiline = true
			return nil
		}

		if isMultiline {
			if !strings.HasSuffix(s, "`") {
				// Multi-line input is not over
				inputBuffer = append(inputBuffer, s)
				return nil
			}
			// Handle last line of multi-line input
			inputBuffer = append(inputBuffer, strings.TrimSuffix(s, "`"))
			isMultiline = false
		} else {
			inputBuffer = append(inputBuffer, s)
		}

		input := strings.Join(inputBuffer, "\n")
		err := handleUserMsg(input)
		if err != nil {
			return err
		}

		inputBuffer = []string{}
		isMultiline = false
		return nil
	}

	var livePrefix = func() (string, bool) {
		if isMultiline {
			return "··· ", true
		}
		return ">>> ", false
	}

	p := prompt.New(
		func(s string) {
			err := executor(s)
			if err != nil {
				os.Exit(0)
			}
		},
		func(document prompt.Document) []prompt.Suggest {
			return nil
		},
		prompt.OptionPrefixTextColor(prompt.Cyan),
		prompt.OptionPrefix(">>> "),
		prompt.OptionLivePrefix(livePrefix),
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlC,
			Fn: func(*prompt.Buffer) {
				os.Exit(0)
			},
		}),
	)

	p.Run()

}

func render(text string) {
	var renderPlainText = func(text string) {
		fmt.Println(text)
	}

	var renderMarkdown = func(text string) {
		out, err := glamour.Render(text, viper.GetString(FlagRenderStyle))
		if err != nil {
			renderPlainText(text)
		}
		fmt.Println(out)
	}

	var renderer func(text string)

	if viper.GetBool(FlagRenderPlainText) {
		renderer = renderPlainText
	} else {
		renderer = renderMarkdown
	}
	renderer(text)
}
