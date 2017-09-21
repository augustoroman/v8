
export PATH := ${PATH}:$(shell pwd)/depot_tools

TARGET := native

lib: v8 include libv8
	GYPFLAGS="-Dv8_use_external_startup_data=0 -Dv8_enable_i18n_support=0 -Dv8_enable_gdbjit=0" make -C v8 ${TARGET} i18nsupport=off
	strip -S v8/out/${TARGET}/libv8_*.a
	cp v8/out/${TARGET}/*.a libv8/
	cp v8/include/*.h include/
	cp -r v8/include/libplatform include/

v8:
	fetch --nohooks v8
	cd v8 && gclient sync

libv8:
	mkdir libv8

include:
	mkdir include
	
depot_tools :
	git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git