package autolinkplugin

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mattermost/mattermost-plugin-autolink/server/autolink"
	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"
)

// Config from config.json
type Config struct {
	EnableAdminCommand bool
	EnableOnUpdate     bool
	Links              []autolink.Autolink
	EnableVisaCard     bool
	EnableMasterCard   bool
	EnableSSN          bool
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	var c Config
	err := p.API.LoadPluginConfiguration(&c)
	if err != nil {
		return err
	}

	c.Links = p.AddPreConfigLinks(&c)

	for i := range c.Links {
		err = c.Links[i].Compile()
		if err != nil {
			mlog.Error(fmt.Sprintf("Error creating autolinker: %+v: %v", c.Links[i], err))
		}
	}

	p.UpdateConfig(func(conf *Config) {
		*conf = c
	})

	go func() {
		if c.EnableAdminCommand {
			_ = p.API.RegisterCommand(&model.Command{
				Trigger:          "autolink",
				DisplayName:      "Autolink",
				Description:      "Autolink administration.",
				AutoComplete:     true,
				AutoCompleteDesc: "Available commands: add, delete, disable, enable, list, set, test",
				AutoCompleteHint: "[command]",
			})
		} else {
			_ = p.API.UnregisterCommand("", "autolink")
		}
	}()

	return nil
}

func (p *Plugin) getConfig() Config {
	p.confLock.RLock()
	defer p.confLock.RUnlock()
	return p.conf
}

func (p *Plugin) GetLinks() []autolink.Autolink {
	p.confLock.RLock()
	defer p.confLock.RUnlock()
	return p.conf.Links
}

func (p *Plugin) SaveLinks(links []autolink.Autolink) error {
	p.UpdateConfig(func(conf *Config) {
		conf.Links = links
	})
	appErr := p.API.SavePluginConfig(p.getConfig().ToConfig())
	if appErr != nil {
		return fmt.Errorf("Unable to save links: %w", appErr)
	}

	return nil
}

// GetPreConFigLinks gets the preconfigured links from the plugin config
func (p *Plugin) AddPreConfigLinks(c *Config) []autolink.Autolink {
	link := autolink.Autolink{
		Name:     "VisaCard",
		Pattern:  "(?P<VISA>(?P<part1>4\\d{3})[ -]?(?P<part2>\\d{4})[ -]?(?P<part3>\\d{4})[ -]?(?P<LastFour>[0-9]{4}))",
		Template: "VISA XXXX-XXXX-XXXX-$LastFour",
		Disabled: !c.EnableVisaCard,
	}
	c.Links = append(c.Links, link)

	link = autolink.Autolink{
		Name:     "MasterCard",
		Pattern:  "(?P<MasterCard]((?P<part1>5[1-5]\\d{2})[ -]?(?P<part2>\\d{4})[ -]?(?P<part3>\\d{4})[ -]?(?P<LastFour>[0-9]{4}))",
		Template: "MasterCard XXXX-XXXX-XXXX-$LastFour",
		Disabled: !c.EnableMasterCard,
	}
	c.Links = append(c.Links, link)

	link = autolink.Autolink{
		Name:     "SSN",
		Pattern:  "(?P<SSN>(?P<part1>\\d{3})[ -]?(?P<part2>\\d{2})[ -]?(?P<LastFour>[0-9]{4}))",
		Template: "XXX-XX-$LastFour",
		Disabled: !c.EnableSSN,
	}
	c.Links = append(c.Links, link)

	return c.Links
}

func (p *Plugin) UpdateConfig(f func(conf *Config)) Config {
	p.confLock.Lock()
	defer p.confLock.Unlock()

	f(&p.conf)
	return p.conf
}

// ToConfig marshals Config into a tree of map[string]interface{} to pass down
// to p.API.SavePluginConfig, otherwise RPC/gob barfs at the unknown type.
func (conf Config) ToConfig() map[string]interface{} {
	links := []interface{}{}
	for _, l := range conf.Links {
		links = append(links, l.ToConfig())
	}
	return map[string]interface{}{
		"EnableAdminCommand": conf.EnableAdminCommand,
		"EnableOnUpdate":     conf.EnableOnUpdate,
		"Links":              links,
		"EnableVisaCard":     conf.EnableVisaCard,
		"EnableMasterCard":   conf.EnableMasterCard,
		"EnableSSN":          conf.EnableSSN,
	}
}

// Sorted returns a clone of the Config, with links sorted alphabetically
func (conf Config) Sorted() Config {
	sorted := conf
	sorted.Links = append([]autolink.Autolink{}, conf.Links...)
	sort.Slice(conf.Links, func(i, j int) bool {
		return strings.Compare(conf.Links[i].DisplayName(), conf.Links[j].DisplayName()) < 0
	})
	return conf
}