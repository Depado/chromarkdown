package main

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"os"

	"github.com/Depado/bfchroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/gobuffalo/packr"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	bf "gopkg.in/russross/blackfriday.v2"
)

// Built in extensions for the markdown renderer
var exts = bf.NoIntraEmphasis | bf.Tables | bf.FencedCode | bf.Autolink |
	bf.Strikethrough | bf.SpaceHeadings | bf.BackslashLineBreak |
	bf.DefinitionLists | bf.Footnotes

// Built in flags for the markdown renderer
var flags = bf.UseXHTML | bf.Smartypants | bf.SmartypantsFractions |
	bf.SmartypantsDashes | bf.SmartypantsLatexDashes | bf.TOC

// GlobCSS is a byte slice containing the style CSS of the renderer
var GlobCSS template.CSS

// render takes a []byte input and runs the mardown render (with the bfchroma
// plugin enabled and with default values)
func render(input []byte) []byte {
	r := bfchroma.NewRenderer(
		bfchroma.WithoutAutodetect(),
		bfchroma.Extend(bf.NewHTMLRenderer(bf.HTMLRendererParameters{Flags: flags})),
		bfchroma.Style("monokai"),
		bfchroma.ChromaOptions(html.WithClasses()),
	)
	if GlobCSS == "" && r.Formatter.Classes {
		b := new(bytes.Buffer)
		if err := r.Formatter.WriteCSS(b, r.Style); err != nil {
			logrus.WithError(err).Warning("Couldn't write CSS")
		}
		GlobCSS = template.CSS(b.String())
	}
	return bf.Run(
		input,
		bf.WithRenderer(r),
		bf.WithExtensions(exts),
	)
}

var rootCmd = &cobra.Command{
	Use:   "chromarkdown [input file]",
	Short: "Chromarkdown is a simple markdown-to-html renderer",
	Long: `Chromarkdown uses a combination of blackfriday and chroma to render an input markdown file.
It generates standalone HTML files that includes fonts, a grid system and extra CSS.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var fd *os.File
		var t *template.Template
		var in []byte

		box := packr.NewBox("./templates")
		if t, err = template.New("output").Parse(box.String("index.tmpl")); err != nil {
			logrus.WithError(err).Fatal("Couldn't parse template")
		}

		if fd, err = os.Create(viper.GetString("output")); err != nil {
			logrus.WithError(err).Fatal("Couldn't create file")
		}
		defer fd.Close()

		if in, err = ioutil.ReadFile(args[0]); err != nil {
			logrus.WithError(err).Fatal("Couldn't read in.md")
		}
		err = t.ExecuteTemplate(fd, "output", map[string]interface{}{
			"title":    viper.GetString("title"),
			"rendered": template.HTML(string(render(in))),
			"css":      GlobCSS,
		})
		if err != nil {
			logrus.WithError(err).Fatal("Couldn't execute template")
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringP("output", "o", "out.html", "specify the path of the output HTML")
	rootCmd.PersistentFlags().StringP("title", "t", "Ouput", "Specify the title of the HTML page")
	viper.BindPFlags(rootCmd.PersistentFlags())
}

func main() {
	rootCmd.Execute()
}
