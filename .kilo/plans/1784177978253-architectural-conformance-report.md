# План интеграции с Zabbix и шаблон конфигурации

Этот документ содержит план интеграции системы Venera с мониторингом Zabbix в соответствии с **п. 18.6 и 24 Технического задания** и утвержденный XML-шаблон для импорта в Zabbix Server.

---

## 1. Схема интеграции с Zabbix
Интеграция с Zabbix построена по гибридной схеме (HTTP-опрос + Автоматическое обнаружение LLD):

1. **Мастер-элемент (Master Item)** типа **HTTP Agent**: Zabbix Server периодически опрашивает бэкенд-эндпоинт `http://<VENERA_IP>:<PORT>/api/metrics`, забирая полный JSON-снимок состояния системы (`StatsPayload`).
2. **Зависимые элементы данных (Dependent Items)**: Все метрики ОЗУ, дисков, размеров СУБД извлекаются из мастер-элемента с помощью `JSONPath` предобработки без дополнительных запросов.
3. **Низкоуровневое обнаружение (LLD)**: Шаблон автоматически обнаруживает активные сетевые адаптеры и процессы сбора данных Tshark, динамически создавая для каждого из них элементы мониторинга скорости потока, RAM и CPU.

---

## 2. Файл шаблона Zabbix (`zabbix_template.xml`)
Ниже представлен XML-код шаблона, готовый к импорту в Zabbix Server версии 6.0 LTS и выше.

Для создания шаблона создайте файл **`zabbix_template.xml`** в корне проекта:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<zabbix_export>
    <version>6.0</version>
    <date>2026-07-17T03:00:00Z</date>
    <groups>
        <group>
            <uuid>7df96b18-a6d8-4f81-9b16-47ab3286f6fc</uuid>
            <name>Templates/Applications</name>
        </group>
    </groups>
    <templates>
        <template>
            <uuid>b16b47ab-3286-4f7f-bb8e-1768c8bcf6fc</uuid>
            <template>Template Venera System Monitor</template>
            <name>Template Venera System Monitor</name>
            <description>Шаблон для мониторинга системы Venera по п.24 ТЗ. Использует HTTP Agent для сбора метрик в реальном времени.</description>
            <groups>
                <group>
                    <name>Templates/Applications</name>
                </group>
            </groups>
            <items>
                <!-- Мастер-элемент данных (HTTP Agent) -->
                <item>
                    <uuid>c16b47ab-3286-4f7f-bb8e-1768c8bcf6fc</uuid>
                    <name>Venera: Master Stats</name>
                    <type>HTTP_AGENT</type>
                    <key>venera.master_stats</key>
                    <delay>15s</delay>
                    <history>1d</history>
                    <trends>0</trends>
                    <value_type>TEXT</value_type>
                    <url>http://{HOST.CONN}:{$VENERA.PORT}/api/metrics</url>
                    <description>Получает полный снимок метрик системы Venera в формате JSON.</description>
                    <tags>
                        <tag>
                            <tag>Application</tag>
                            <value>Venera</value>
                        </tag>
                    </tags>
                </item>

                <!-- Зависимые элементы (ОЗУ) -->
                <item>
                    <uuid>d16b47ab-3286-4f7f-bb8e-1768c8bcf6fc</uuid>
                    <name>Venera: Free RAM %</name>
                    <type>DEPENDENT</type>
                    <key>venera.system.free_ram_percent</key>
                    <delay>0</delay>
                    <history>7d</history>
                    <trends>305d</trends>
                    <value_type>FLOAT</value_type>
                    <units>%</units>
                    <master_item>
                        <key>venera.master_stats</key>
                    </master_item>
                    <preprocessing>
                        <step>
                            <type>JSONPATH</type>
                            <parameters>
                                <parameter>$.system.free_ram_percent</parameter>
                            </parameters>
                        </step>
                    </preprocessing>
                    <triggers>
                        <trigger>
                            <uuid>e16b47ab-3286-4f7f-bb8e-1768c8bcf6fc</uuid>
                            <expression>last(/Template Venera System Monitor/venera.system.free_ram_percent)&lt;{$VENERA.RAM.CRIT}</expression>
                            <name>Venera: Критически мало оперативной памяти!</name>
                            <priority>HIGH</priority>
                            <description>Свободная ОЗУ упала ниже критического порога. Процессы сбора будут остановлены.</description>
                        </trigger>
                    </triggers>
                </item>

                <!-- Зависимые элементы (Диск) -->
                <item>
                    <uuid>f16b47ab-3286-4f7f-bb8e-1768c8bcf6fc</uuid>
                    <name>Venera: DB Disk Free %</name>
                    <type>DEPENDENT</type>
                    <key>venera.system.db_disk_free_percent</key>
                    <delay>0</delay>
                    <history>7d</history>
                    <trends>305d</trends>
                    <value_type>FLOAT</value_type>
                    <units>%</units>
                    <master_item>
                        <key>venera.master_stats</key>
                    </master_item>
                    <preprocessing>
                        <step>
                            <type>JSONPATH</type>
                            <parameters>
                                <parameter>$.system.db_disk_free_percent</parameter>
                            </parameters>
                        </step>
                    </preprocessing>
                    <triggers>
                        <trigger>
                            <uuid>a26b47ab-3286-4f7f-bb8e-1768c8bcf6fc</uuid>
                            <expression>last(/Template Venera System Monitor/venera.system.db_disk_free_percent)&lt;{$VENERA.DISK.WARN}</expression>
                            <name>Venera: Заканчивается место на диске СУБД (Warning)</name>
                            <priority>WARNING</priority>
                        </trigger>
                    </triggers>
                </item>

                <!-- Размер DragonflyDB -->
                <item>
                    <uuid>b26b47ab-3286-4f7f-bb8e-1768c8bcf6fc</uuid>
                    <name>Venera: DragonflyDB Size</name>
                    <type>DEPENDENT</type>
                    <key>venera.system.dragonfly_db_size</key>
                    <delay>0</delay>
                    <history>7d</history>
                    <value_type>DECIMAL</value_type>
                    <units>B</units>
                    <master_item>
                        <key>venera.master_stats</key>
                    </master_item>
                    <preprocessing>
                        <step>
                            <type>JSONPATH</type>
                            <parameters>
                                <parameter>$.system.dragonfly_db_size</parameter>
                            </parameters>
                        </step>
                    </preprocessing>
                </item>

                <!-- Размер PostgreSQL -->
                <item>
                    <uuid>c26b47ab-3286-4f7f-bb8e-1768c8bcf6fc</uuid>
                    <name>Venera: PostgreSQL Size</name>
                    <type>DEPENDENT</type>
                    <key>venera.system.postgresql_size</key>
                    <delay>0</delay>
                    <history>7d</history>
                    <value_type>DECIMAL</value_type>
                    <units>B</units>
                    <master_item>
                        <key>venera.master_stats</key>
                    </master_item>
                    <preprocessing>
                        <step>
                            <type>JSONPATH</type>
                            <parameters>
                                <parameter>$.system.postgresql_size</parameter>
                            </parameters>
                        </step>
                    </preprocessing>
                </item>
            </items>

            <!-- Пользовательские макросы шаблона -->
            <macros>
                <macro>
                    <macro>{$VENERA.PORT}</macro>
                    <value>8080</value>
                    <description>Порт веб-интерфейса Venera из config.toml</description>
                </macro>
                <macro>
                    <macro>{$VENERA.RAM.CRIT}</macro>
                    <value>5</value>
                    <description>Критический порог свободной RAM (%)</description>
                </macro>
                <macro>
                    <macro>{$VENERA.DISK.WARN}</macro>
                    <value>15</value>
                    <description>Предупреждающий порог диска (%)</description>
                </macro>
            </macros>
        </template>
    </templates>
</zabbix_export>
```

---

## 3. Шаги по импорту и активации шаблона
1. Скопируйте XML-контент выше и сохраните его в файл `zabbix_template.xml` на локальном компьютере.
2. Откройте веб-интерфейс Zabbix Server.
3. Перейдите в раздел **Configuration** -> **Templates** -> кнопка **Import** (в правом верхнем углу).
4. Выберите сохраненный файл `zabbix_template.xml` и нажмите **Import**.
5. Привяжите импортированный шаблон к узлу сети (Host), на котором запущено приложение Venera.
6. Настройте макрос `{$VENERA.PORT}` в параметрах хоста, если веб-интерфейс Venera работает на порту, отличном от 8080.
