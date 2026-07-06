-- 1. 如果数据库不存在则创建
CREATE DATABASE IF NOT EXISTS `smart_reserve` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `smart_reserve`;

-- 2. 自动创建预约订单表
CREATE TABLE IF NOT EXISTS `appointment_order`
(
    `id`          bigint unsigned  NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `user_id`     bigint unsigned  NOT NULL DEFAULT '0' COMMENT '用户ID',
    `schedule_id` bigint unsigned  NOT NULL DEFAULT '0' COMMENT '场次ID',
    `order_no`    varchar(64)      NOT NULL DEFAULT '' COMMENT '预约流水号',
    `status`      tinyint unsigned NOT NULL DEFAULT '1' COMMENT '状态 1:预约成功 2:已取消',
    `create_time` timestamp        NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time` timestamp        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_order_no` (`order_no`),
    KEY `idx_user_schedule` (`user_id`, `schedule_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='预约订单表';

-- 3. 创建场次表
CREATE TABLE IF NOT EXISTS `schedule`
(
    `id`          bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '场次ID',
    `title`       varchar(128)    NOT NULL DEFAULT '' COMMENT '场次名称(如：上午场、XX签证预约)',
    `total_slots` int             NOT NULL DEFAULT '0' COMMENT '总名额数量',
    `start_time`  datetime        NOT NULL COMMENT '预约开始时间',
    `create_time` timestamp       NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='预约场次表';

-- 4. 预埋一条种子数据：99号场次，总共 100 个名额，开始时间定在 2026 年
INSERT INTO `schedule` (`id`, `title`, `total_slots`, `start_time`)
VALUES (99, '2026暑期高峰签证预约-特别专场', 100, '2026-08-01 09:00:00')
ON DUPLICATE KEY UPDATE `total_slots` = 100;