package elrondapi

import (
	"bytes"
	"math/big"

	twos "github.com/ElrondNetwork/big-int-util/twos-complement"
	"github.com/ElrondNetwork/wasm-vm/arwen"
	"github.com/ElrondNetwork/wasm-vm/executor"
	"github.com/ElrondNetwork/wasm-vm/math"
)

const (
	mBufferNewName                = "mBufferNew"
	mBufferNewFromBytesName       = "mBufferNewFromBytes"
	mBufferGetLengthName          = "mBufferGetLength"
	mBufferGetBytesName           = "mBufferGetBytes"
	mBufferGetByteSliceName       = "mBufferGetByteSlice"
	mBufferCopyByteSliceName      = "mBufferCopyByteSlice"
	mBufferEqName                 = "mBufferEq"
	mBufferSetBytesName           = "mBufferSetBytes"
	mBufferAppendName             = "mBufferAppend"
	mBufferAppendBytesName        = "mBufferAppendBytes"
	mBufferToBigIntUnsignedName   = "mBufferToBigIntUnsigned"
	mBufferToBigIntSignedName     = "mBufferToBigIntSigned"
	mBufferFromBigIntUnsignedName = "mBufferFromBigIntUnsigned"
	mBufferFromBigIntSignedName   = "mBufferFromBigIntSigned"
	mBufferStorageStoreName       = "mBufferStorageStore"
	mBufferStorageLoadName        = "mBufferStorageLoad"
	mBufferGetArgumentName        = "mBufferGetArgument"
	mBufferFinishName             = "mBufferFinish"
	mBufferSetRandomName          = "mBufferSetRandom"
	mBufferToBigFloatName         = "mBufferToBigFloat"
	mBufferFromBigFloatName       = "mBufferFromBigFloat"
)

// ManagedBufferImports populates imports with the ManagedBuffer API methods
func ManagedBufferImports(imports executor.ImportFunctionReceiver) error {
	imports.Namespace("env")

	return nil
}

// MBufferNew EIInterface implementation
func (context *EICallbacks) MBufferNew() int32 {
	managedType := context.GetManagedTypesContext()
	metering := context.GetMeteringContext()

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferNew
	metering.UseGasAndAddTracedGas(mBufferNewName, gasToUse)

	return managedType.NewManagedBuffer()
}

// MBufferNewFromBytes EIInterface implementation
func (context *EICallbacks) MBufferNewFromBytes(dataOffset int32, dataLength int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferNewFromBytes
	metering.UseGasAndAddTracedGas(mBufferNewFromBytesName, gasToUse)

	data, err := runtime.MemLoad(dataOffset, dataLength)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return -1
	}

	return managedType.NewManagedBufferFromBytes(data)
}

// MBufferGetLength EIInterface implementation
func (context *EICallbacks) MBufferGetLength(mBufferHandle int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferGetLength
	metering.UseGasAndAddTracedGas(mBufferGetLengthName, gasToUse)

	length := managedType.GetLength(mBufferHandle)
	if length == -1 {
		_ = context.WithFault(arwen.ErrNoManagedBufferUnderThisHandle, runtime.ManagedBufferAPIErrorShouldFailExecution())
		return -1
	}

	return length
}

// MBufferGetBytes EIInterface implementation
func (context *EICallbacks) MBufferGetBytes(mBufferHandle int32, resultOffset int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()
	metering.StartGasTracing(mBufferGetBytesName)

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferGetBytes
	metering.UseAndTraceGas(gasToUse)

	mBufferBytes, err := managedType.GetBytes(mBufferHandle)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}
	managedType.ConsumeGasForBytes(mBufferBytes)

	err = runtime.MemStore(resultOffset, mBufferBytes)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}

	return 0
}

// MBufferGetByteSlice EIInterface implementation
func (context *EICallbacks) MBufferGetByteSlice(sourceHandle int32, startingPosition int32, sliceLength int32, resultOffset int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()
	metering.StartGasTracing(mBufferGetByteSliceName)

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferGetByteSlice
	metering.UseAndTraceGas(gasToUse)

	sourceBytes, err := managedType.GetBytes(sourceHandle)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}
	managedType.ConsumeGasForBytes(sourceBytes)

	if startingPosition < 0 || sliceLength < 0 || int(startingPosition+sliceLength) > len(sourceBytes) {
		// does not fail execution if slice exceeds bounds
		return 1
	}

	slice := sourceBytes[startingPosition : startingPosition+sliceLength]
	err = runtime.MemStore(resultOffset, slice)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}

	return 0
}

// MBufferCopyByteSlice EIInterface implementation
func (context *EICallbacks) MBufferCopyByteSlice(sourceHandle int32, startingPosition int32, sliceLength int32, destinationHandle int32) int32 {
	host := context.GetVMHost()
	return ManagedBufferCopyByteSliceWithHost(host, sourceHandle, startingPosition, sliceLength, destinationHandle)
}

func ManagedBufferCopyByteSliceWithHost(host arwen.VMHost, sourceHandle int32, startingPosition int32, sliceLength int32, destinationHandle int32) int32 {
	managedType := host.ManagedTypes()
	runtime := host.Runtime()
	metering := host.Metering()
	metering.StartGasTracing(mBufferCopyByteSliceName)

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferCopyByteSlice
	metering.UseAndTraceGas(gasToUse)

	sourceBytes, err := managedType.GetBytes(sourceHandle)
	if WithFaultAndHost(host, err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}
	managedType.ConsumeGasForBytes(sourceBytes)

	if startingPosition < 0 || sliceLength < 0 || int(startingPosition+sliceLength) > len(sourceBytes) {
		// does not fail execution if slice exceeds bounds
		return 1
	}

	slice := sourceBytes[startingPosition : startingPosition+sliceLength]
	managedType.SetBytes(destinationHandle, slice)

	gasToUse = math.MulUint64(metering.GasSchedule().BaseOperationCost.DataCopyPerByte, uint64(len(slice)))
	metering.UseAndTraceGas(gasToUse)

	return 0
}

// MBufferEq EIInterface implementation
func (context *EICallbacks) MBufferEq(mBufferHandle1 int32, mBufferHandle2 int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()
	metering.StartGasTracing(mBufferEqName)

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferCopyByteSlice
	metering.UseAndTraceGas(gasToUse)

	bytes1, err := managedType.GetBytes(mBufferHandle1)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return -1
	}
	managedType.ConsumeGasForBytes(bytes1)

	bytes2, err := managedType.GetBytes(mBufferHandle2)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return -1
	}
	managedType.ConsumeGasForBytes(bytes2)

	if bytes.Equal(bytes1, bytes2) {
		return 1
	}

	return 0
}

// MBufferSetBytes EIInterface implementation
func (context *EICallbacks) MBufferSetBytes(mBufferHandle int32, dataOffset int32, dataLength int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()
	metering.StartGasTracing(mBufferSetBytesName)

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferSetBytes
	metering.UseAndTraceGas(gasToUse)

	data, err := runtime.MemLoad(dataOffset, dataLength)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}
	managedType.ConsumeGasForBytes(data)
	managedType.SetBytes(mBufferHandle, data)

	return 0
}

// MBufferSetByteSlice EIInterface implementation
func (context *EICallbacks) MBufferSetByteSlice(mBufferHandle int32, startingPosition int32, dataLength int32, dataOffset int32) int32 {
	host := context.GetVMHost()
	return ManagedBufferSetByteSliceWithHost(host, mBufferHandle, startingPosition, dataLength, dataOffset)
}

func ManagedBufferSetByteSliceWithHost(host arwen.VMHost, mBufferHandle int32, startingPosition int32, dataLength int32, dataOffset int32) int32 {
	runtime := host.Runtime()
	metering := host.Metering()
	metering.StartGasTracing(mBufferGetByteSliceName)

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferSetBytes
	metering.UseAndTraceGas(gasToUse)

	data, err := runtime.MemLoad(dataOffset, dataLength)
	if WithFaultAndHost(host, err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}

	return ManagedBufferSetByteSliceWithTypedArgs(host, mBufferHandle, startingPosition, dataLength, data)
}

func ManagedBufferSetByteSliceWithTypedArgs(host arwen.VMHost, mBufferHandle int32, startingPosition int32, dataLength int32, data []byte) int32 {
	managedType := host.ManagedTypes()
	runtime := host.Runtime()
	metering := host.Metering()
	metering.StartGasTracing(mBufferGetByteSliceName)

	managedType.ConsumeGasForBytes(data)

	bufferBytes, err := managedType.GetBytes(mBufferHandle)
	if WithFaultAndHost(host, err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}

	if startingPosition < 0 || dataLength < 0 || int(startingPosition+dataLength) > len(bufferBytes) {
		// does not fail execution if slice exceeds bounds
		return 1
	}

	start := int(startingPosition)
	length := int(dataLength)
	destination := bufferBytes[start : start+length]

	copy(destination, data)

	managedType.SetBytes(mBufferHandle, bufferBytes)

	return 0
}

// MBufferAppend EIInterface implementation
func (context *EICallbacks) MBufferAppend(accumulatorHandle int32, dataHandle int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()
	metering.StartGasTracing(mBufferAppendName)

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferAppend
	metering.UseAndTraceGas(gasToUse)

	dataBufferBytes, err := managedType.GetBytes(dataHandle)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}
	managedType.ConsumeGasForBytes(dataBufferBytes)

	isSuccess := managedType.AppendBytes(accumulatorHandle, dataBufferBytes)
	if !isSuccess {
		_ = context.WithFault(arwen.ErrNoManagedBufferUnderThisHandle, runtime.ManagedBufferAPIErrorShouldFailExecution())
		return 1
	}

	return 0
}

// MBufferAppendBytes EIInterface implementation
func (context *EICallbacks) MBufferAppendBytes(accumulatorHandle int32, dataOffset int32, dataLength int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()
	metering.StartGasTracing(mBufferAppendBytesName)

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferAppendBytes
	metering.UseAndTraceGas(gasToUse)

	data, err := runtime.MemLoad(dataOffset, dataLength)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}

	isSuccess := managedType.AppendBytes(accumulatorHandle, data)
	if !isSuccess {
		_ = context.WithFault(arwen.ErrNoManagedBufferUnderThisHandle, runtime.ManagedBufferAPIErrorShouldFailExecution())
		return 1
	}

	gasToUse = math.MulUint64(metering.GasSchedule().BaseOperationCost.DataCopyPerByte, uint64(len(data)))
	metering.UseAndTraceGas(gasToUse)

	return 0
}

// MBufferToBigIntUnsigned EIInterface implementation
func (context *EICallbacks) MBufferToBigIntUnsigned(mBufferHandle int32, bigIntHandle int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferToBigIntUnsigned
	metering.UseGasAndAddTracedGas(mBufferToBigIntUnsignedName, gasToUse)

	managedBuffer, err := managedType.GetBytes(mBufferHandle)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}

	bigInt := managedType.GetBigIntOrCreate(bigIntHandle)
	bigInt.SetBytes(managedBuffer)

	return 0
}

// MBufferToBigIntSigned EIInterface implementation
func (context *EICallbacks) MBufferToBigIntSigned(mBufferHandle int32, bigIntHandle int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferToBigIntSigned
	metering.UseGasAndAddTracedGas(mBufferToBigIntSignedName, gasToUse)

	managedBuffer, err := managedType.GetBytes(mBufferHandle)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}

	bigInt := managedType.GetBigIntOrCreate(bigIntHandle)
	twos.SetBytes(bigInt, managedBuffer)

	return 0
}

// MBufferFromBigIntUnsigned EIInterface implementation
func (context *EICallbacks) MBufferFromBigIntUnsigned(mBufferHandle int32, bigIntHandle int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferFromBigIntUnsigned
	metering.UseGasAndAddTracedGas(mBufferFromBigIntUnsignedName, gasToUse)

	value, err := managedType.GetBigInt(bigIntHandle)
	if context.WithFault(err, runtime.BigIntAPIErrorShouldFailExecution()) {
		return 1
	}

	managedType.SetBytes(mBufferHandle, value.Bytes())

	return 0
}

// MBufferFromBigIntSigned EIInterface implementation
func (context *EICallbacks) MBufferFromBigIntSigned(mBufferHandle int32, bigIntHandle int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferFromBigIntSigned
	metering.UseGasAndAddTracedGas(mBufferFromBigIntSignedName, gasToUse)

	value, err := managedType.GetBigInt(bigIntHandle)
	if context.WithFault(err, runtime.BigIntAPIErrorShouldFailExecution()) {
		return 1
	}

	managedType.SetBytes(mBufferHandle, twos.ToBytes(value))
	return 0
}

// MBufferToBigFloat EIInterface implementation
func (context *EICallbacks) MBufferToBigFloat(mBufferHandle, bigFloatHandle int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()
	metering.StartGasTracing(mBufferToBigFloatName)

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferToBigFloat
	metering.UseAndTraceGas(gasToUse)

	managedBuffer, err := managedType.GetBytes(mBufferHandle)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}

	managedType.ConsumeGasForBytes(managedBuffer)
	if managedType.EncodedBigFloatIsNotValid(managedBuffer) {
		_ = context.WithFault(arwen.ErrBigFloatWrongPrecision, runtime.BigFloatAPIErrorShouldFailExecution())
		return 1
	}

	value, err := managedType.GetBigFloatOrCreate(bigFloatHandle)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}

	bigFloat := new(big.Float)
	err = bigFloat.GobDecode(managedBuffer)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}
	if bigFloat.IsInf() {
		_ = context.WithFault(arwen.ErrInfinityFloatOperation, runtime.BigFloatAPIErrorShouldFailExecution())
		return 1
	}

	value.Set(bigFloat)
	return 0
}

// MBufferFromBigFloat EIInterface implementation
func (context *EICallbacks) MBufferFromBigFloat(mBufferHandle, bigFloatHandle int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()
	metering.StartGasTracing(mBufferFromBigFloatName)

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferFromBigFloat
	metering.UseAndTraceGas(gasToUse)

	value, err := managedType.GetBigFloat(bigFloatHandle)
	if context.WithFault(err, runtime.BigFloatAPIErrorShouldFailExecution()) {
		return 1
	}

	encodedFloat, err := value.GobEncode()
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}
	managedType.ConsumeGasForBytes(encodedFloat)

	managedType.SetBytes(mBufferHandle, encodedFloat)

	return 0
}

// MBufferStorageStore EIInterface implementation
func (context *EICallbacks) MBufferStorageStore(keyHandle int32, sourceHandle int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	storage := context.GetStorageContext()
	metering := context.GetMeteringContext()

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferStorageStore
	metering.UseGasAndAddTracedGas(mBufferStorageStoreName, gasToUse)

	key, err := managedType.GetBytes(keyHandle)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}

	sourceBytes, err := managedType.GetBytes(sourceHandle)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}

	_, err = storage.SetStorage(key, sourceBytes)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}

	return 0
}

// MBufferStorageLoad EIInterface implementation
func (context *EICallbacks) MBufferStorageLoad(keyHandle int32, destinationHandle int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	storage := context.GetStorageContext()
	metering := context.GetMeteringContext()

	key, err := managedType.GetBytes(keyHandle)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}

	storageBytes, usedCache := storage.GetStorage(key)
	storage.UseGasForStorageLoad(mBufferStorageLoadName, metering.GasSchedule().ManagedBufferAPICost.MBufferStorageLoad, usedCache)

	managedType.SetBytes(destinationHandle, storageBytes)

	return 0
}

// MBufferStorageLoadFromAddress EIInterface implementation
func (context *EICallbacks) MBufferStorageLoadFromAddress(addressHandle, keyHandle, destinationHandle int32) {
	host := context.GetVMHost()
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()

	key, err := managedType.GetBytes(keyHandle)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return
	}

	address, err := managedType.GetBytes(addressHandle)
	if err != nil {
		_ = context.WithFault(arwen.ErrArgOutOfRange, runtime.ElrondAPIErrorShouldFailExecution())
		return
	}

	storageBytes := StorageLoadFromAddressWithTypedArgs(host, address, key)

	managedType.SetBytes(destinationHandle, storageBytes)
}

// MBufferGetArgument EIInterface implementation
func (context *EICallbacks) MBufferGetArgument(id int32, destinationHandle int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferGetArgument
	metering.UseGasAndAddTracedGas(mBufferGetArgumentName, gasToUse)

	args := runtime.Arguments()
	if int32(len(args)) <= id || id < 0 {
		context.WithFault(arwen.ErrArgOutOfRange, runtime.ElrondAPIErrorShouldFailExecution())
		return 1
	}
	managedType.SetBytes(destinationHandle, args[id])
	return 0
}

// MBufferFinish EIInterface implementation
func (context *EICallbacks) MBufferFinish(sourceHandle int32) int32 {
	managedType := context.GetManagedTypesContext()
	output := context.GetOutputContext()
	metering := context.GetMeteringContext()
	runtime := context.GetRuntimeContext()
	metering.StartGasTracing(mBufferFinishName)

	gasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferFinish
	metering.UseAndTraceGas(gasToUse)

	sourceBytes, err := managedType.GetBytes(sourceHandle)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return 1
	}

	gasToUse = math.MulUint64(metering.GasSchedule().BaseOperationCost.PersistPerByte, uint64(len(sourceBytes)))
	err = metering.UseGasBounded(gasToUse)
	if err != nil {
		_ = context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution())
		return 1
	}

	output.Finish(sourceBytes)
	return 0
}

// MBufferSetRandom EIInterface implementation
func (context *EICallbacks) MBufferSetRandom(destinationHandle int32, length int32) int32 {
	managedType := context.GetManagedTypesContext()
	runtime := context.GetRuntimeContext()
	metering := context.GetMeteringContext()

	if length < 1 {
		_ = context.WithFault(arwen.ErrLengthOfBufferNotCorrect, runtime.ManagedBufferAPIErrorShouldFailExecution())
		return -1
	}

	baseGasToUse := metering.GasSchedule().ManagedBufferAPICost.MBufferSetRandom
	lengthDependentGasToUse := math.MulUint64(metering.GasSchedule().BaseOperationCost.DataCopyPerByte, uint64(length))
	gasToUse := math.AddUint64(baseGasToUse, lengthDependentGasToUse)
	metering.UseGasAndAddTracedGas(mBufferSetRandomName, gasToUse)

	randomizer := managedType.GetRandReader()
	buffer := make([]byte, length)
	_, err := randomizer.Read(buffer)
	if context.WithFault(err, runtime.ManagedBufferAPIErrorShouldFailExecution()) {
		return -1
	}

	managedType.SetBytes(destinationHandle, buffer)
	return 0
}
