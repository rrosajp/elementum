package api

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/anacrolix/missinggo/perf"
	"github.com/asdine/storm/q"
	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"

	"github.com/elgatito/elementum/bittorrent"
	"github.com/elgatito/elementum/config"
	"github.com/elgatito/elementum/database"
	"github.com/elgatito/elementum/exit"
	"github.com/elgatito/elementum/library"
	"github.com/elgatito/elementum/proxy"
	"github.com/elgatito/elementum/tmdb"
	"github.com/elgatito/elementum/util"
	"github.com/elgatito/elementum/xbmc"
)

// Changelog display
func Changelog(ctx *gin.Context) {
	defer perf.ScopeTimer()()

	xbmcHost, _ := xbmc.GetXBMCHost(ctx.ClientIP())

	changelogPath := filepath.Join(config.Get().Info.Path, "whatsnew.txt")
	if _, err := os.Stat(changelogPath); err != nil {
		ctx.String(404, err.Error())
		return
	}

	title := "LOCALIZE[30355]"
	text, err := os.ReadFile(changelogPath)
	if err != nil {
		ctx.String(404, err.Error())
		return
	}

	xbmcHost.DialogText(title, string(text))
	ctx.String(200, "")
}

// Donate display
func Donate(ctx *gin.Context) {
	xbmcHost, _ := xbmc.GetXBMCHost(ctx.ClientIP())

	xbmcHost.Dialog("Elementum", "LOCALIZE[30141]")
	ctx.String(200, "")
}

// Settings display
func Settings(ctx *gin.Context) {
	xbmcHost, _ := xbmc.GetXBMCHost(ctx.ClientIP())

	addon := ctx.Params.ByName("addon")
	if addon == "" {
		addon = "plugin.video.elementum"
	}

	xbmcHost.AddonSettings(addon)
	ctx.String(200, "")
}

// Status display
func Status(ctx *gin.Context) {
	defer perf.ScopeTimer()()

	xbmcHost, _ := xbmc.GetXBMCHost(ctx.ClientIP())

	title := "LOCALIZE[30393]"
	text := ""

	text += `[B]LOCALIZE[30394]:[/B] %s

[B]LOCALIZE[30395]:[/B] %s
[B]LOCALIZE[30396]:[/B] %d
[B]LOCALIZE[30488]:[/B] %d

[COLOR pink][B]LOCALIZE[30399]:[/B][/COLOR]
    [B]LOCALIZE[30397]:[/B] %s
    [B]LOCALIZE[30401]:[/B] %s
    [B]LOCALIZE[30439]:[/B] %s
    [B]LOCALIZE[30398]:[/B] %s

[COLOR pink][B]LOCALIZE[30400]:[/B][/COLOR]
    [B]LOCALIZE[30403]:[/B] %s
    [B]LOCALIZE[30402]:[/B] %s

    [B]LOCALIZE[30404]:[/B] %d
    [B]LOCALIZE[30405]:[/B] %d
    [B]LOCALIZE[30458]:[/B] %d
    [B]LOCALIZE[30459]:[/B] %d
`

	ip := "127.0.0.1"
	if localIP, err := util.LocalIP(); err == nil {
		ip = localIP.String()
	}

	port := config.Args.LocalPort
	webAddress := fmt.Sprintf("http://%s:%d/web", ip, port)
	debugAllAddress := fmt.Sprintf("http://%s:%d/debug/all", ip, port)
	debugBundleAddress := fmt.Sprintf("http://%s:%d/debug/bundle", ip, port)
	infoAddress := fmt.Sprintf("http://%s:%d/info", ip, port)

	appSize := fileSize(filepath.Join(config.Get().Info.Profile, database.GetStorm().GetFilename()))
	cacheSize := fileSize(filepath.Join(config.Get().Info.Profile, database.GetCache().GetFilename()))

	torrentsCount, _ := database.GetStormDB().Count(&database.TorrentAssignMetadata{})
	queriesCount, _ := database.GetStormDB().Count(&database.QueryHistory{})
	deletedMoviesCount, _ := database.GetStormDB().Select(q.Eq("MediaType", library.MovieType), q.Eq("State", library.StateDeleted)).Count(&database.LibraryItem{})
	deletedShowsCount, _ := database.GetStormDB().Select(q.Eq("MediaType", library.ShowType), q.Eq("State", library.StateDeleted)).Count(&database.LibraryItem{})

	text = fmt.Sprintf(text,
		util.GetVersion(),
		ip,
		port,
		proxy.ProxyPort,

		webAddress,
		infoAddress,
		debugAllAddress,
		debugBundleAddress,

		appSize,
		cacheSize,

		torrentsCount,
		queriesCount,
		deletedMoviesCount,
		deletedShowsCount,
	)

	xbmcHost.DialogText(title, string(text))
	ctx.String(200, "")
}

func fileSize(path string) string {
	fi, err := os.Stat(path)
	if err != nil {
		return ""
	}

	return humanize.Bytes(uint64(fi.Size()))
}

// SelectNetworkInterface ...
func SelectNetworkInterface(ctx *gin.Context) {
	xbmcHost, _ := xbmc.GetXBMCHost(ctx.ClientIP())

	typeName := ctx.Params.ByName("type")

	ifaces, err := net.Interfaces()
	if err != nil {
		ctx.String(404, err.Error())
		return
	}

	items := make([]string, 0, len(ifaces))

	for _, i := range ifaces {
		name := i.Name
		address := ""

		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			v4 := ip.To4()
			if v4 != nil {
				address = v4.String()
			}
		}

		if address != "" {
			name = fmt.Sprintf("[B]%s[/B] (%s)", i.Name, address)
		} else {
			name = fmt.Sprintf("[B]%s[/B]", i.Name)
		}

		items = append(items, name)
	}

	choice := xbmcHost.ListDialog("LOCALIZE[30474]", items...)
	if choice >= 0 {
		xbmcHost.SetSetting("listen_autodetect_ip", "false")
		if typeName == "listen" {
			xbmcHost.SetSetting("listen_interfaces", ifaces[choice].Name)
		} else {
			xbmcHost.SetSetting("outgoing_interfaces", ifaces[choice].Name)
		}
	}

	ctx.String(200, "")
}

// SelectStrmLanguage ...
func SelectStrmLanguage(ctx *gin.Context) {
	xbmcHost, _ := xbmc.GetXBMCHost(ctx.ClientIP())

	items := make([]string, 0)
	items = append(items, xbmcHost.GetLocalizedString(30477))

	languages := tmdb.GetLanguages(config.Get().Language)
	for _, l := range languages {
		items = append(items, l.Name)
	}

	choice := xbmcHost.ListDialog("LOCALIZE[30373]", items...)
	if choice >= 1 {
		xbmcHost.SetSetting("strm_language", languages[choice-1].Name+" | "+languages[choice-1].Iso639_1)
	} else if choice == 0 {
		xbmcHost.SetSetting("strm_language", "Original")
	}

	ctx.String(200, "")
}

func Reload(s *bittorrent.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer perf.ScopeTimer()()

		s.Reconfigure()
		ctx.String(200, "true")
	}
}

func Restart(shutdown func(code int)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer perf.ScopeTimer()()

		ctx.String(200, "true")
		shutdown(exit.ExitCodeRestart)
	}
}

func Shutdown(shutdown func(code int)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer perf.ScopeTimer()()

		ctx.String(200, "true")
		go shutdown(exit.ExitCodeSuccess)
	}
}
