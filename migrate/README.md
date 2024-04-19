# Migrate 数据库迁移工具

## 一、说明

本包为日常工作中用到的数据库迁移管理工具提炼而来，本包仅提供基础的迁移管理实现，具体的嵌入到命令行中请自主实现或嵌入即可。

## 二、使用示例

````
// 实例化迁移工具
migration := migrate.New(migrate.Config{
    Dir:       "migrations", // 迁移文件的存储目录，相对于main包或binary可执行文件
    TableName: "migrations", // 迁移工具本身所依赖的Db数据表表名
    DB:        db,           // *sql.DB 对象实例
})

// 创建迁移文件
migration.Create('不带后缀的文件名称')

// 查看迁移状态
migration.Status()

// 执行迁移
migration.ExecUp()

// 回滚迁移
migration.ExecDown("给空字符串则回滚最后1条，给定迁移文件名称则仅回滚指定名称的迁移")
````

migrate实例对象提供的方法可以嵌入到cli命令行入口，通过不同的参数执行不同的迁移命令

## 三、迁移文件写法

迁移文件本身就是一个SQL文件，基础结构如下，其中`-- +migrate Up`、`-- +migrate Down`是不可缺少的标识符。

`migration.Create('不带后缀的文件名称')`被调用时将自动在`migrate.Config.Dir`目录生成如下结构的sql文件。

`-- +migrate Up`下方写迁移被ExecUp时也就是创建时执行的sql语句

`-- +migrate Down`下方写迁移被ExecDown时也就是回滚时执行的sql语句

多行或者复杂的sql语句，可使用`-- +migrate StatementBegin`和`-- +migrate StatementEnd`作为标识符进行包裹，这个包裹符并不是必须的。

> 注意上述标识符本身是一个标准的SQL语句中的注释段，注意其中的空格、`+`、大小写都是刻意如此设定的。

空迁移文件样例：
````
-- +migrate Up


-- +migrate Down

````

无包裹符迁移文件写法样例：

````
-- +migrate Up
CREATE TABLE migrate_1 (id int);
CREATE TABLE migrate_2 (id int);

-- +migrate Down
DROP TABLE migrate_1;
DROP TABLE migrate_2;
DROP TABLE test_table;
````

有包裹符迁移文件写法样例：
````
-- +migrate Up
-- +migrate StatementBegin
CREATE TABLE `test_table` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='test_table';
-- +migrate StatementEnd

-- +migrate Down
DROP TABLE migrate_1;
DROP TABLE migrate_2;
DROP TABLE test_table;
````
