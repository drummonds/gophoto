# gophoto

GoKrazy picture frame

Shows pictures from various sources eg photoprism album

paneldemo.go produces a demo panel as a png

## Development

- Convert code to build framebuffer to export png
    - _Get margin to work_ ✅
    - _get image to display_ ✅
    - simplify panel build
    - create demo code?
- _Show an panel on browser locally_ ✅
- Show on browser in panel
- Show on framebuffer
- _Get image from photoprism_ ✅
    - photoprism in gocrazy

Want:
- 32bit or 64 bit bits per pixel rather than 16 in 556 pattern
- To allow you to choose which album
- To choose audience
- Auto select audience
- To store current location in album so can pick up after reset
- To only increment when TV is one
    - could use remote control
    - or EDID HDMI info
- To allow users to not see an image - needs concept of viewer/audience
- To choose which picture you want to promote or demote
- Make transition easier
- colocate photoprism and DB
- use cockroach db on same box?
- auto find photoprism



## Notes

### Raspberry Pi power supply

The Raspberry Pi on an inadequte power supply will cause the Pi to brownout and reset.  Using the
Raspberry Pi 5 on the Raspberry pi psu is rock solid so far.

- 
