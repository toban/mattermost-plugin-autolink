package autolink

import (
	"fmt"
	"regexp"
	"strings"

	"io/ioutil"
	"log"
	"net/http"
)

// Autolink represents a pattern to autolink.
type Autolink struct {
	Name                 string
	Disabled             bool
	Pattern              string
	Template             string
	LookupUrlTemplate    string
	Scope                []string
	WordMatch            bool
	DisableNonWordPrefix bool
	DisableNonWordSuffix bool

	template          string
	lookupUrlTemplate string
	re                *regexp.Regexp
	canReplaceAll     bool
}

func (l Autolink) Equals(x Autolink) bool {
	if l.Disabled != x.Disabled ||
		l.DisableNonWordPrefix != x.DisableNonWordPrefix ||
		l.DisableNonWordSuffix != x.DisableNonWordSuffix ||
		l.Name != x.Name ||
		l.Pattern != x.Pattern ||
		len(l.Scope) != len(x.Scope) ||
		l.Template != x.Template ||
		l.WordMatch != x.WordMatch {
		return false
	}
	for i, scope := range l.Scope {
		if scope != x.Scope[i] {
			return false
		}
	}
	return true
}

// DisplayName returns a display name for the link.
func (l Autolink) DisplayName() string {
	if l.Name != "" {
		return l.Name
	}
	return l.Pattern
}

// Compile compiles the link's regular expression
func (l *Autolink) Compile() error {
	if l.Disabled || len(l.Pattern) == 0 || len(l.Template) == 0 {
		return nil
	}

	// `\b` can be used with ReplaceAll since it does not consume characters,
	// custom patterns can not and need to be processed one at a time.
	canReplaceAll := false
	pattern := l.Pattern
	template := l.Template
	// todo do something smart to this
	lookupUrlTemplate := l.LookupUrlTemplate

	if !l.DisableNonWordPrefix {
		if l.WordMatch {
			pattern = `\b` + pattern
			canReplaceAll = true
		} else {
			pattern = `(?P<MattermostNonWordPrefix>(^|\s))` + pattern
			template = `${MattermostNonWordPrefix}` + template
		}
	}
	if !l.DisableNonWordSuffix {
		if l.WordMatch {
			pattern += `\b`
			canReplaceAll = true
		} else {
			pattern += `(?P<MattermostNonWordSuffix>$|[\s\.\!\?\,\)])`
			template += `${MattermostNonWordSuffix}`
		}
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	l.re = re
	l.template = template
	l.lookupUrlTemplate = lookupUrlTemplate
	l.canReplaceAll = canReplaceAll

	return nil
}

func doReq(url string) (content string) {

	resp, err := http.Get(url)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	return string(body)
}

func getTitle(content string) (title string) {

	re := regexp.MustCompile("<title>(.*)</title>")

	parts := re.FindStringSubmatch(content)

	if len(parts) > 0 {
		return parts[1]
	} else {
		return "no title"
	}
}

func Var_dump(expression ...interface{}) {
	fmt.Println(fmt.Sprintf("%#v", expression))
}

// Replace will subsitute the regex's with the supplied links
func (l Autolink) Replace(message string) string {
	if l.re == nil {
		return message
	}

	// Since they don't consume, `\b`s require no special handling, can just ReplaceAll
	if l.canReplaceAll {

		// lookup happens here if the template is set
		//if l.lookupUrlTemplate != "" {
		lookupUrl := l.re.ReplaceAllString(message, l.lookupUrlTemplate)
		content := doReq(lookupUrl)
		title := getTitle(content)
		return fmt.Sprintf("[%s](%s)", title, lookupUrl)
		//}

		//return l.re.ReplaceAllString(message, l.template)
	}

	// Replace one at a time
	in := []byte(message)
	out := []byte{}
	for {
		if len(in) == 0 {
			break
		}
		// get index of first occurance of Txxxxx ex: I found T298595, and T123456
		submatch := l.re.FindSubmatchIndex(in)
		if submatch == nil {
			break
		}

		// TODO Add a cache for lookups
		log.Println("--------------------------------------------------------")

		// asdf T298595
		// submatch = []int{4, 12, 4, 5, 4, 5, 5, 12, 12, 12}
		Var_dump(submatch)
		submatchWord := string(in[submatch[0]:submatch[1]])
		//log.Println("https://phabricator.wikimedia.org/" + submatchWord)
		// asdf
		out = append(out, in[:submatch[0]]...)
		// message from index 0 up until the point of T298595
		//log.Println(string(in[:submatch[0]]))

		lookupUrl := l.re.ReplaceAllString(strings.TrimSpace(submatchWord), l.lookupUrlTemplate)
		content := doReq(lookupUrl)
		title := getTitle(content)
		markupLink := fmt.Sprintf("[%s](%s)", title, lookupUrl)
		titleByteArray := []byte(markupLink)
		//log.Println(title)

		// fmt.Sprintf("[%s](%s)", title, lookupUrl)
		// replaces submatch with template in the entire thing
		//out = l.re.Expand(out, []byte(l.template), in, submatch)
		out = append(out, titleByteArray...)

		log.Println(string(out))

		in = in[submatch[1]:]
	}
	out = append(out, in...)
	return string(out)
}

// ToMarkdown prints a Link as a markdown list element
func (l Autolink) ToMarkdown(i int) string {
	text := "- "
	if i > 0 {
		text += fmt.Sprintf("%v: ", i)
	}
	if l.Name != "" {
		if l.Disabled {
			text += fmt.Sprintf("~~%s~~", l.Name)
		} else {
			text += l.Name
		}
	}
	if l.Disabled {
		text += " **Disabled**"
	}
	text += "\n"

	text += fmt.Sprintf("  - Pattern: `%s`\n", l.Pattern)
	text += fmt.Sprintf("  - Template: `%s`\n", l.Template)

	if l.DisableNonWordPrefix {
		text += fmt.Sprintf("  - DisableNonWordPrefix: `%v`\n", l.DisableNonWordPrefix)
	}
	if l.DisableNonWordSuffix {
		text += fmt.Sprintf("  - DisableNonWordSuffix: `%v`\n", l.DisableNonWordSuffix)
	}
	if len(l.Scope) != 0 {
		text += fmt.Sprintf("  - Scope: `%v`\n", l.Scope)
	}
	if l.WordMatch {
		text += fmt.Sprintf("  - WordMatch: `%v`\n", l.WordMatch)
	}
	return text
}

// ToConfig returns a JSON-encodable Link represented solely with map[string]
// interface and []string types, compatible with gob/RPC, to be used in
// SavePluginConfig
func (l Autolink) ToConfig() map[string]interface{} {
	return map[string]interface{}{
		"Name":                 l.Name,
		"Pattern":              l.Pattern,
		"Template":             l.Template,
		"Scope":                l.Scope,
		"DisableNonWordPrefix": l.DisableNonWordPrefix,
		"DisableNonWordSuffix": l.DisableNonWordSuffix,
		"WordMatch":            l.WordMatch,
		"Disabled":             l.Disabled,
	}
}
