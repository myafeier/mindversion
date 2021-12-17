package mindvision

/*
#cgo linux,!android CFLAGS: -I../mvsdk/include
#cgo linux,!android LDFLAGS: -L${SRCDIR}/../mvsdk/lib -lMVSDK -Wl -rpath=./mvsdk/lib
#cgo darwin CFLAGS: -I../mvsdk/include
#cgo darwin LDFLAGS: -L${SRCDIR}/../mvsdk/lib -lmvsdk
#include "CameraApi.h"
#include <stdio.h>
CameraHandle handle;
*/
import "C"
import (
	"fmt"
	"log"
	"time"
	"unsafe"
	//"github.com/myafeier/log"
)

func init() {
	log.SetPrefix("[GoMindVersion]")
	log.SetFlags(log.Llongfile | log.Ltime)
}

type Camera struct {
	devices  [32]C.tSdkCameraDevInfo
	bufsize  int
	filepath string
}

func (s *Camera) Init(filepath string) {
	status := C.CameraSdkInit(C.int(0))
	err := sdkError(status)
	if err != nil {
		panic(err)
	}
	if filepath == "" {
		s.filepath = "./"
	} else {
		s.filepath = filepath
	}
}

func (s *Camera) UnInit() {
	C.CameraUnInit(C.handle)
}

// 查看设备列表
func (s *Camera) EnumerateDevice() (list []*Device, err error) {
	var count int = 32
	// CameraEnumerateDevice 要求传入数组指针，及数组长度指针
	status := C.CameraEnumerateDevice((*C.tSdkCameraDevInfo)(unsafe.Pointer(&(s.devices[0]))), (*C.int)(unsafe.Pointer(&count)))

	err = sdkError(status)
	if err != nil {
		log.Printf(err.Error())
		return
	}
	for i := 0; i < count; i++ {
		t := new(Device)
		t.ParseC(s.devices[i])
		list = append(list, t)
	}

	return
}

// 选择并激活相机
func (s *Camera) ActiveCamera(idx int, exposeSecond float32) (err error) {
	status := C.CameraInit((*C.tSdkCameraDevInfo)(unsafe.Pointer(&(s.devices[idx]))), C.int(-1), C.int(-1), (*C.CameraHandle)(unsafe.Pointer(&C.handle)))
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}

	// 获取相机的参数
	var capability C.tSdkCameraCapbility

	status = C.CameraGetCapability(C.handle, (*C.tSdkCameraCapbility)(unsafe.Pointer(&capability)))
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	if int(capability.sIspCapacity.bMonoSensor) == 1 {
		s.bufsize = int(capability.sResolutionRange.iWidthMax * capability.sResolutionRange.iHeightMax)
		status = C.CameraSetIspOutFormat(C.handle, C.CAMERA_MEDIA_TYPE_MONO8)

	} else {
		s.bufsize = int(capability.sResolutionRange.iWidthMax*capability.sResolutionRange.iHeightMax) * 3
		status = C.CameraSetIspOutFormat(C.handle, C.CAMERA_MEDIA_TYPE_BGR8)
	}

	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	// 相机模式切换成连续采集
	status = C.CameraSetTriggerMode(C.handle, C.int(0))
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}

	// 手动曝光模式,并设置曝光时间
	status = C.CameraSetAeState(C.handle, C.int(0))
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	//曝光3秒
	status = C.CameraSetExposureTime(C.handle, C.double(exposeSecond*1000000))
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	//让SDK进入工作模式
	status = C.CameraPlay(C.handle)
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	log.Println("设备已就绪，等待指令")
	return
}

// 获取一张图片
func (s *Camera) Grab() (fn string, err error) {
	// 分配RGB buffer，用来存放ISP输出的图像
	//备注：从相机传输到PC端的是RAW数据，在PC端通过软件ISP转为RGB数据（如果是黑白相机就不需要转换格式，但是ISP还有其它处理，所以也需要分配这个buffer）

	//log.Printf("bufsize: %d\n", s.bufsize)
	outputPtr := C.CameraAlignMalloc(C.int(s.bufsize), 4)
	defer C.CameraAlignFree(outputPtr)

	var frameInfo C.tSdkFrameHead
	rawDataPtr := C.CameraAlignMalloc(C.int(s.bufsize), 4)
	status := C.CameraGetImageBuffer(C.handle, (*C.tSdkFrameHead)(unsafe.Pointer(&frameInfo)), (**C.BYTE)(unsafe.Pointer(&rawDataPtr)), 6000)
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	status = C.CameraImageProcess(C.handle, rawDataPtr, outputPtr, (*C.tSdkFrameHead)(unsafe.Pointer(&frameInfo)))
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	//	blob := C.GoBytes(unsafe.Pointer(outputPtr), C.int(s.bufsize))

	//fmt.Printf("head: %v\n,blob:%v\n", frameInfo, blob)
	status = C.CameraReleaseImageBuffer(C.handle, rawDataPtr)
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	fn = fmt.Sprintf(s.filepath+"%d.bmp", time.Now().UnixNano())
	status = C.CameraSaveImage(C.handle, C.CString(fn), outputPtr, (*C.tSdkFrameHead)(unsafe.Pointer(&frameInfo)), C.FILE_BMP, 0)
	err = sdkError(status)
	if err != nil {
		log.Println(err.Error())
		return
	}
	return
}
