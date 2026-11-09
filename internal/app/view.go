package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (a *App) view(c *ui.Context) {
	th := c.Theme()
	ui.Column(c).Grow(1).Background(th.Background).Children(func() {
		a.welcomeView(c)
	})
}

func (a *App) welcomeView(c *ui.Context) {
	th := c.Theme()
	ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Padding(16).Children(func() {
		ui.Column(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Folcon").FontSize(22).Bold()
			ui.Text(c, "把应用图标设为所在文件夹的图标。").
				FontSize(12).TextColor(th.TextMuted)

			if a.running {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					if a.busy {
						ui.Spinner(c)
					}
					text := a.status
					if text == "" {
						text = "正在处理…"
					}
					ui.Text(c, text).FontSize(12).TextColor(th.TextMuted)
				})
				ui.Text(c, fmt.Sprintf("第 %d 个，共 %d 个 · 已应用 %d 个",
					a.index+1, len(a.folders), a.applied)).
					FontSize(11).TextColor(th.TextMuted)
			}

			ui.Column(c).Padding(12).Radius(10).Border(1, th.Border).
				Background(th.Surface).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(c, "把文件夹拖到这里").FontSize(13).Bold()
				ui.PrimaryButton(c, "选择文件夹…").OnClick(a.chooseFolders).Disabled(a.running)
			})

			if a.notice != "" {
				ui.Text(c, a.notice).FontSize(11).TextColor(th.TextMuted)
			}
		})
	})
}
