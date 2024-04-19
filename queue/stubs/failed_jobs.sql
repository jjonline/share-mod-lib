-- 失败任务存储表，可用于后续重试
CREATE TABLE `prefix_failed_jobs` (
     `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
     `connection` text NOT NULL,
     `queue` text NOT NULL,
     `payload` longtext NOT NULL,
     `exception` longtext NOT NULL,
     `failed_at` int(10) unsigned NOT NULL,
     PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
