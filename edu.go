
package main

import (
	"crypto/md5"  //md5加解密
	"crypto/rand"  //生成随机数
	"encoding/base64"  //base64的编码和解码
	"encoding/hex"  //16进制
	"encoding/json"  //json存储
	"errors"   //错误处理
	"fmt"
	"io"  //读写
	"strconv"  //数值类型转换
	"time"  //获取当前时间

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type SimpleChaincode struct {
}

//创建 一个 Buffer （其实底层就是一个 []byte， 字节切片）

var BackGroundNo int = 0
var RecordNo int = 0

type School struct {
	Name           string
	Location       string
	Address        string
	StudentAddress []string
}

type Student struct {
	Name         string
	Address      string
	BackgroundId []int
}

// 学历信息，当离开学校才能记入
type Background struct {
	Id       int
	ExitTime int64
	Sch  School
	Status   string //0:毕业 1：退学
}

type Record struct {
	Id              int
	SchoolAddress   string
	StudentAddress  string
	SchoolSign      string
	ModifyTime      int64
	ModifyOperation string // 0:正常毕业 1：退学 2:入学
}


func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

/*
 *CORE_PEER_ADDRESS=peer:7051 CORE_CHAINCODE_ID_NAME=mycc:0 ./chaincode004
 *sch add:07bab4bb6c4f53ffa0c64f1814651cc，a60669235a1700f4df5484dfc0d04b0e,zhang jinjia add:50d479525aadaee036552e120d176181
 *peer chaincode install -p chaincodedev/chaincode/chaincode004 -n mycc -v 0
 *peer chaincode instantiate -n mycc -v 0 -c '{"Args":["init"]}' -C myc
 *peer chaincode invoke -n mycc -c '{"Args":["createSchool", "NJUPT", "Nanjing"]}' -C myc
 *peer chaincode invoke -n mycc -c '{"Args":["createStudent", "Zhang Jinjia"]}' -C myc
 *peer chaincode invoke -n mycc -c '{"Args":["getSchoolSignByAddress", "sch add"]}' -C myc
 *peer chaincode invoke -n mycc -c '{"Args":["enrollStudent", "sch address", "sch sign", "stu add"]}' -C myc
 *peer chaincode invoke -n mycc -c '{"Args":["updateDiploma", "sch add", "sch sign", "stu add", "0"]}' -C myc
 *peer chaincode invoke -n mycc -c '{"Args":["getStudentByAddress", "stu add"]}' -C myc
 *peer chaincode invoke -n mycc -c '{"Args":["getSchoolByAddress", "sch add"]}' -C myc
 *peer chaincode invoke -n mycc -c '{"Args":["delSchoolByAddress", "sch add"]}' -C myc
 *peer chaincode invoke -n mycc -c '{"Args":["delStudentByAddress", "stu add"]}' -C myc
 *peer chaincode invoke -n mycc -c '{"Args":["getRecords"]}' -C myc
 */
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	// 获取请求调用智能合约的方法和参数
	function, args := stub.GetFunctionAndParameters()

	if function == "createSchool" {
		return t.createSchool(stub, args)
	} else if function == "createStudent" {
		return t.createStudent(stub, args)
	} else if function == "getSchoolSignByAddress" {
		return t.getSchoolSignByAddress(stub,args)
	} else if function == "enrollStudent" {
		return t.enrollStudent(stub, args)
	} else if function == "updateDiploma" {
		return t.updateDiploma(stub, args)
	} else if function == "getRecords" {
		return t.getRecords(stub)
	} else if function == "getRecordById" {
		return t.getRecordById(stub, args)
	} else if function == "getStudentByAddress" {
		return t.getStudentByAddress(stub, args)
	} else if function == "getSchoolByAddress" {
		return t.getSchoolByAddress(stub, args)
	} else if function == "getBackgroundById" {
		return t.getBackgroundById(stub, args)
	} else if function == "delSchoolByAdress"{
		return t.delSchoolByAdress(stub,args)
	} else if function == "delStudentByAdress" {
		return t.delStudentByAdress(stub,args)
	}

	return shim.Success(nil)
}

/*
 * 添加一所新学校
 * args[0] 学校名称
 * args[1] 学校所在位置
 */
func (t *SimpleChaincode) createSchool(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	var school School
	var schoolBytes []byte
	var stuAddress []string
	var address string
	//生成学校的地址
	address = GetAddress()

	school = School{Name: args[0], Location: args[1], Address: address, StudentAddress: stuAddress}
	//将学校信息写入账本中
	err := writeSchool(stub, school)
	if err != nil {
		shim.Error("Error write school")
	}

	schoolBytes, err = json.Marshal(school)
	if err != nil {
		return shim.Error("Error retrieving schoolBytes")
	}
	//返回添加的学校信息
	return shim.Success(schoolBytes)
}

/*
 * 添加一名新学生
 * args[0] 学生的姓名
 */
func (t *SimpleChaincode) createStudent(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	var student Student
	var studentBytes []byte
	var stuAddress string
	var bgID []int
	//生成学生的地址
	stuAddress = GetAddress()

	student = Student{Name: args[0], Address: stuAddress, BackgroundId: bgID}
	//将学生的信息写入账本
	err := writeStudent(stub, student)
	if err != nil {
		return shim.Error("Error write student")
	}

	studentBytes, err = json.Marshal(student)
	if err != nil {
		return shim.Error("Error retrieving studentBytes")
	}
	//返回添加的学生信息
	return shim.Success(studentBytes)
}

/*
 * 学校招生（返回学校信息）
 * args[0] 学校账户地址
 * args[1] 学校签名
 * args[2] 学生账户地址
 */
func (t *SimpleChaincode) enrollStudent(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	schAddress := args[0]
	schoolSign := args[1]
	stuAddress := args[2]

	var school School
	var schBytes []byte
	var err error

	// 根据学校账户地址获取学校信息
	schBytes, err = stub.GetState(schAddress)
	if err != nil {
		return shim.Error("Error retrieving data")
	}
	//将json对象转化成school对象
	err = json.Unmarshal(schBytes, &school)
	if err != nil {
		return shim.Error("Error unmarshalling data")
	}

	var record Record
	record = Record{Id: RecordNo, SchoolAddress: schAddress, StudentAddress: stuAddress, SchoolSign: schoolSign, ModifyTime: time.Now().Unix(), ModifyOperation: "2"} // 2 表示入学
	//将record信息写入账本中
	err = writeRecord(stub, record)
	if err != nil {
		return shim.Error("Error write record")
	}

	school.StudentAddress = append(school.StudentAddress, stuAddress)
	//更新学校的信息
	err = writeSchool(stub, school)
	if err != nil {
		return shim.Error("Error write school")
	}

	RecordNo = RecordNo + 1  //记录+1

	recordBytes, err := json.Marshal(record)
	if err != nil {
		return shim.Error("Error retrieving recordBytes")
	}
	//返回record信息
	return shim.Success(recordBytes)
}


/*
 *通过地址来删除学校
 *args[0] address
 */
func (t *SimpleChaincode)delSchoolByAdress(stub shim.ChaincodeStubInterface,args []string) pb.Response {
	schAddress := args[0]
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	err := stub.DelState(schAddress)
	if err != nil {
		return shim.Error("Failed to delete state: " + err.Error())
	}
	fmt.Println("- end delete_School")
	return shim.Success(nil)

}

/*
 *通过地址来删除学生，通过ID来删除相关record记录
 *args[0] address
 *args[1] recID
 */
func (t *SimpleChaincode) delStudentByAdress(stub shim.ChaincodeStubInterface,args []string) pb.Response {
	stuAddress := args[0]
	recID := args[1]
	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	err := stub.DelState(stuAddress)
	if err != nil {
		return shim.Error("Failed to delete state: " + err.Error())
	}

	err = stub.DelState("Record" + recID)
	if err != nil {
		return shim.Error("Failed to delete state: " + err.Error())
	}

	fmt.Println("- end delete_Student")
	return shim.Success(nil)
}

/*
 * 由学校更新学生学历信息，并签名（返回记录信息）
 * args[0] 学校账户地址
 * args[1] 学校签名
 * args[2] 待修改学生的账户地址
 * args[3] 对该学生的学历进行怎样的修改，0：正常毕业  1：退学
 */
func (t *SimpleChaincode) updateDiploma(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4")
	}

	schAddress := args[0]
	schoolSign := args[1]
	stuAddress := args[2]
	modOperation := args[3]

	var recordBytes []byte
	var student Student
	var stuBytes []byte
	var err error

	// 根据学生账户地址获取学生信息
	stuBytes, err = stub.GetState(stuAddress)

	if err != nil {
		return shim.Error("Error retrieving data")
	}
	err = json.Unmarshal(stuBytes, &student)
	if err != nil {
		return shim.Error("Error unmarshalling data")
	}

	var record Record
	record = Record{Id: RecordNo, SchoolAddress: schAddress, StudentAddress: stuAddress, SchoolSign: schoolSign, ModifyTime: time.Now().Unix(), ModifyOperation: modOperation}
	//更新record
	err = writeRecord(stub, record)
	if err != nil {
		return shim.Error("Error write record")
	}


	var school School
	var schBytes []byte

	// 根据学校账户地址获取学校信息
	schBytes, err = stub.GetState(schAddress)
	if err != nil {
		return shim.Error("Error retrieving data")
	}
	//将json对象转化成school对象
	err = json.Unmarshal(schBytes, &school)
	if err != nil {
		return shim.Error("Error unmarshalling data")
	}




	var background Background
	background = Background{Id: BackGroundNo, ExitTime: time.Now().Unix(), Sch: school, Status: modOperation}

	//将backgroud写入账本
	err = writeBackground(stub, background)
	if err != nil {
		return shim.Error("Error write background")
	}

	// 如果学生正常毕业，也要更新学生的教育背景
	if modOperation == "0" {
		student.BackgroundId = append(student.BackgroundId, BackGroundNo)
		student = Student{Name: student.Name, Address: student.Address, BackgroundId: student.BackgroundId}
		err = writeStudent(stub, student)
		if err != nil {
			return shim.Error("Error write student")
		}
	}

	BackGroundNo = BackGroundNo + 1
	recordBytes, err = json.Marshal(record)
	if err != nil {
		return shim.Error("Error retrieving schoolBytes")
	}

   //返回record信息
	return shim.Success(recordBytes)
}

/*
 * 通过学生的地址获取学生的学历信息
 * args[0] address
 */
func (t *SimpleChaincode) getStudentByAddress(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	stuBytes, err := stub.GetState(args[0])
	if err != nil {
		shim.Error("Error retrieving data")
	}
	return shim.Success(stuBytes)
}


/*
 *通过地址生成学校的签名并返回                t
 *args[0] address
 */
func (t *SimpleChaincode)getSchoolSignByAddress(stub shim.ChaincodeStubInterface,args []string) pb.Response {
	schAddress := args[0]
	if len(args) != 1{
		return  shim.Error("Incorrect number of arguments. Expecting 1")
	}
	schoolSign := GetAddress()

//	str := []string{schAddress,schoolSign}

//	schBytes := "\x00" + strings.Join(str, "\x20\x00")

  	fmt.Printf("学校的地址：%s，学校的签名：%s\n",schAddress, schoolSign)
	//return shim.Success([]byte(schBytes))
	return shim.Success(nil)
}
/*
 * 通过地址获取学校的信息
 * args[0] address
 */
func (t *SimpleChaincode) getSchoolByAddress(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	schBytes, err := stub.GetState(args[0])
	if err != nil {
		shim.Error("Error retrieving data")
	}
	return shim.Success(schBytes)
}

/*
 * 通过 Id 获取记录
 * args[0] 记录的 Id
 */
func (t *SimpleChaincode) getRecordById(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	recBytes, err := stub.GetState("Record" + args[0])
	if err != nil {
		return shim.Error("Error retrieving data")
	}

	return shim.Success(recBytes)
}

/*
 * 获取全部记录（如果记录数大于10,返回前10个）
 */
func (t *SimpleChaincode) getRecords(stub shim.ChaincodeStubInterface) pb.Response {
	var records []Record
	var number string
	var err error
	var record Record
	var recBytes []byte
	if RecordNo < 10 {
		i := 0
		for i <= RecordNo {
			number = strconv.Itoa(i)
			recBytes, err = stub.GetState("Record" + number)
			if err != nil {
				return shim.Error("Error get detail")
			}
			err = json.Unmarshal(recBytes, &record)
			if err != nil {
				return shim.Error("Error unmarshalling data")
			}
			records = append(records, record)
			i = i + 1
		}
	} else {
		i := 0
		for i < 10 {
			number = strconv.Itoa(i)
			recBytes, err = stub.GetState("Record" + number)
			if err != nil {
				return shim.Error("Error get detail")
			}
			err = json.Unmarshal(recBytes, &record)
			if err != nil {
				return shim.Error("Error unmarshalling data")
			}
			records = append(records, record)
			i = i + 1
		}
	}
	recordsBytes, err := json.Marshal(records)
	if err != nil {
		shim.Error("Error get records")
	}
	return shim.Success(recordsBytes)
}

/*
 * 通过 Id 获取所存储的学历信息
 * args[0] ID
 */
func (t *SimpleChaincode) getBackgroundById(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	backBytes, err := stub.GetState("BackGround" + args[0])
	if err != nil {
		return shim.Error("Error retrieving data")
	}
	return shim.Success(backBytes)
}

func writeRecord(stub shim.ChaincodeStubInterface, record Record) error {
	var recID string

	recordBytes, err := json.Marshal(record)
	if err != nil {
		return err
	}

	recID = strconv.Itoa(record.Id)
	//写入账本,k:"Record"+recID,v: recordBytes
	err = stub.PutState("Record"+recID, recordBytes)
	if err != nil {
		return errors.New("PutState Error" + err.Error())
	}
	return nil
}

func writeSchool(stub shim.ChaincodeStubInterface, school School) error {
	//将school对象序列化为json格式
	schBytes, err := json.Marshal(school)
	if err != nil {
		return err
	}
	//将学校的地址和创建的学校相关信息写入区块链
	err = stub.PutState(school.Address, schBytes)
	if err != nil {
		return errors.New("PutState Error" + err.Error())
	}
	return nil
}

func writeStudent(stub shim.ChaincodeStubInterface, student Student) error {
	stuBytes, err := json.Marshal(student)
	if err != nil {
		return err
	}

	err = stub.PutState(student.Address, stuBytes)
	if err != nil {
		return errors.New("PutState Error" + err.Error())
	}
	return nil
}

func writeBackground(stub shim.ChaincodeStubInterface, background Background) error {
	var backID string
	backBytes, err := json.Marshal(background)
	if err != nil {
		return err
	}

	backID = strconv.Itoa(background.Id)
	err = stub.PutState("BackGround"+backID, backBytes)
	if err != nil {
		return errors.New("PutState Error" + err.Error())
	}


	return nil
}

/*
 * 生成Address
 */
func GetAddress() (string) {
	var address string
	b := make([]byte, 48)

	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}

	h := md5.New()
	h.Write([]byte(base64.URLEncoding.EncodeToString(b)))

	address = hex.EncodeToString(h.Sum(nil))

	return address
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}


}
