# rm output/*; \
libcamera-vid \
	--camera 0 \
	--width 640 \
	--height 480 \
	--framerate 5 \
	 --flush 1 \
	--save-pts timestamps.txt \
	-t 0 \
	--shutter 10 \
	--gain 1 \
	--ev 0 \
	--denoise off \
	--contrast 1 \
	--listen \
	--inline \
	--codec h264 \
	-o tcp://0.0.0.0:8888 \
| ./go.neose-fsp-camera.gocv-driver; \
cat timestamps.txt | wc -l && cat timestamps.txt | tail -n 1
