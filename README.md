# Sitdown

Hacking our standing desks for evil

This is a Pandora Hackathon project aimed at controlling our neighbors' standing desks with
Rasberry Pis. Disclaimer: code mostly written over the course of two days.

## Raspberry Pi 3 v1.2b setup

1. Remove `console=serial0...` from /boot/cmdline.txt
2. Add `enable_uart=1` and `dtoverlay=pi3-disable-bt` to /boot/config.txt
3. Move sitdown.service to /lib/systemd/system
4. Create a file at /home/pi/id.conf and put an indentifier string in there
4. Sync this repository and run `go install`, moving the resulting `sitdown` binary to /usr/bin
5. Run `sudo systemctl enable sitdown.service` and `sudo systemctl start sitdown.service`
