# Sitdown

Hacking our standing desks for evil

## Raspberry Pi 3 v1.2b setup

1. Remove `console=serial0...` from /boot/cmdline.txt
2. Add `enable_uart=1` and `dtoverlay=pi3-disable-bt` to /boot/config.txt
3. Move sitdown.service to /lib/systemd/system
4. Run `sudo systemctl enable sitdown.service` and `sudo systemctl start sitdown.service`
