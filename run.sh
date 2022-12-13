rm output/*; \
libcamera-raw \
	--camera 0 \
	--width 640 \
	--height 480 \
	--framerate 30 \
	 --flush 1 \
	--save-pts timestamps.txt \
	-t 1000 \
	--shutter 10 \
	--gain 1 \
	--ev 0 \
	--vflip 1 \
	--denoise off \
	--contrast 1 \
	-o - \
| ./go.neose-fsp-camera.gocv-driver; \
cat timestamps.txt | wc -l && cat timestamps.txt | tail -n 1
