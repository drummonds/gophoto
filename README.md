# gophoto

GoKrazy picture frame

Shows pictures on the Linux frame buffer (HDMI on a Raspberry Pi) from
various sources, currently a PhotoPrism album. Runs as a
[gokrazy](https://gokrazy.org/) appliance, so it is pure Go with no cgo.

## Layout

| Path | Purpose |
|---|---|
| `gophoto.go` | The appliance: paints the frame buffer and serves a status page |
| `internal/frame` | `PictureFrame`, its panels and the PhotoPrism image source |
| `internal/fb`, `internal/linuxvt` | Linux frame buffer and virtual terminal ioctls |
| `internal/drawing` | Scaling and colour helpers |
| `cmd/paneldemo`, `cmd/prisimdemo` | Render a frame to `framebuffer.png` without hardware |
| `cmd/x11*Demo` | Show the frame in an X11 window on a desktop |

## Development

Standard tasks: `task check` runs fmt, vet and test. `task local` runs the
appliance against this machine's frame buffer.

Deploy with `task gok:update` (network update of a running gokrazy
instance) or `task gok:image` (full image to `x.img`). The gokrazy instance
name is the `GOK_NAME` variable in `Taskfile.yml`.

PhotoPrism connection details are read from the environment by
`internal/frame/photoprism.go`.

## Notes

### Raspberry Pi power supply

The Raspberry Pi on an inadequate power supply will cause the Pi to brownout
and reset. Using the Raspberry Pi 5 on the Raspberry Pi PSU is rock solid so
far.

## Links

| | |
|---|---|
| Documentation | https://gophoto.docs.bytestone.uk/ |
| Source (Forgejo) | https://git.bytestone.uk/hum3/gophoto |
| Mirror (GitHub) | https://github.com/drummonds/gophoto |
