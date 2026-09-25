# gophoto

A [gokrazy](https://gokrazy.org/) picture frame for the Raspberry Pi. It
paints photos from a PhotoPrism album straight onto the Linux frame buffer,
so a Pi plugged into a TV becomes a digital photo frame with no desktop or
browser in between.

## How it works

The appliance opens the frame buffer and the virtual terminal, builds a
`PictureFrame` sized to the display, and renders it into an intermediate
buffer that is copied to the frame buffer on each tick. The frame is made of
panels; the photo panel scales each image to cover the display and advances
through the album on a timer. A small status page is served over HTTP for
checking the appliance from the LAN.

Desktop demos render the same frame to a PNG or an X11 window so layout can
be worked on without a Pi.

## Pages

- [README](README.html): layout, development and deployment
- [Changelog](CHANGELOG.html)
- [Roadmap](ROADMAP.html)

## Links

- Source: https://git.bytestone.uk/hum3/gophoto
- Mirror: https://github.com/drummonds/gophoto
