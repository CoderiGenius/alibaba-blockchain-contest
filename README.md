# alibaba blockchain contest
比赛地址：https://tianchi.aliyun.com/competition/information.htm?spm=5176.100069.5678.2.17406e51Mc5z9i&raceId=231665


## 初赛试题

![image](https://work.alibaba-inc.com/aliwork_tfs/g01_alibaba-inc_com/tfscom/TB1yQLDrOOYBuNjSsD4XXbSkFXa.tfsprivate.jpg)

### 在自有系统中搭建Hyperledger Fabric开发测试环境。
推荐软件版本：Hyperledger Fabric v1.1，go version go1.9，Docker version 1.13.1  


### Hyperledger Fabric环境准备及智能合约开发可参考：
```
http://hyperledger-fabric.readthedocs.io/en/release-1.1/chaincode4ade.html
```

### 开发一份基于Golang的chaincode，实现个人履历的存证，功能点如下：

#### 1. 评测系统调用chaincode的addRecord方法，以个人ID为Key，以年份，就读学校/在职公司，学位/职位这三个信息的组合为Value，将履历记录Key-Value写入账本。假设对于同一个ID，同一年份只会写入一条记录，也不会对重复年份的情况进行评测。请注意chaincode的参数接收顺序，调用示例如下：
```
peer chaincode invoke …
-c '{"Args":["addRecord","1001","1999","college1","bachelor"]}'
peer chaincode invoke …
-c '{"Args":["addRecord","1001", "2003","institute1","master"]}'
peer chaincode invoke …
-c '{"Args":["addRecord","1001", "2006","corp1", "engineer"]}'
```
#### 2. 评测系统调用chaincode的getRecord方法，以个人ID和年份为参数，查询出对应的就读学校/在职公司。请注意chaincode的参数接收顺序，调用示例如下（本例应返回"institute1"）：

```
peer chaincode query … -c '{"Args":["getRecord","1001", "2003"]}
```

#### 3. 评测系统调用chaincode的encRecord方法，以个人ID为Key，以年份，就读学校/在职公司，学位／职位为Value，通过transient传入密钥（ENCKEY）和初始化向量（IV），将加密后的履历记录Key-Value写入账本。假设对于同一个ID，同一年份只会写入一条记录，也不会对重复年份的情况进行评测。ENCKEY和IV由评测系统生成，请注意chaincode的参数接收顺序，调用示例如下：

````
peer chaincode invoke …
-c '{"Args":["encRecord","1009","2002","college2","bachelor"]}'
--transient"{\"ENCKEY\":\"$ENCKEY\",\"IV\":\"$IV\"}"
peer chaincode invoke …
-c '{"Args":["encRecord","1009","2006","corp2", "engineer"]}'
--transient"{\"ENCKEY\":\"$ENCKEY\",\"IV\":\"$IV\"}"
peer chaincode invoke …
-c '{"Args":["encRecord","1009","2012","corp3", "manager"]}'
--transient "{\"ENCKEY\":\"$ENCKEY\",\"IV\":\"$IV\"}"
````

#### 4. 评测系统调用chaincode的decRecord方法，以个人ID和年份为参数，通过transient传入密钥(DECKEY)，将对应的就读学校/在职公司解密后返回。DECKEY由评测系统生成，请注意chaincode的参数接收顺序，调用示例如下（本例应正确返回"corp2"）：

````
peer chaincode query ... -c '{"Args":["decRecord", "1009", "2006"]}
--transient"{\"DECKEY\":\"$DECKEY\"}"
````

#### chaincode的Init方法无需对账本内容做初始化

### 评分标准

#### 得分点1（10分）：chaincode可被安装初始化

#### 得分点2（20分）：成功调用addRecord方法，写入若干条记录

#### 得分点3（30分）：成功调用getRecord方法，获得某人某年对应的就读学校/在职公司

#### 得分点4（20分）：成功调用encRecord方法，写入若干条加密记录

#### 得分点5（20分）：成功调用decRecord方法，获得某人某年对应的就读学校/在职公司解密信息

![image](https://work.alibaba-inc.com/aliwork_tfs/g01_alibaba-inc_com/tfscom/TB1cnQ.jXooBKNjSZFPXXXa2XXa.tfsprivate.png)
