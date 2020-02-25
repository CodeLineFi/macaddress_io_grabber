def read_ip_address(machine)
    command = "ip a | grep 'inet' | grep -v '127.0.0.1' | cut -d: -f2"\
              " | awk '{ print $2 }' | cut -f1 -d\"/\""
    result  = ""

    puts "Processing #{ machine.name } ... "

    begin
        machine.communicate.sudo(command) do |type, data|
            result << data if type == :stdout
        end
    rescue
        result = "# NOT-UP"
    end

    result.chomp.split("\n").last
end

def provision(machine, dir, settings)

    confDir = "#{ settings["sync_to"] }/.vagrant/bootstrap/config"
    scriptDir = dir + "/script"

    machine.vm.provision "shell", :path => scriptDir + "/install.sh"

    machine.vm.provision "shell",
        :path => scriptDir + "/config.sh",
        :args => [confDir, settings["sync_to"], settings["db_password"], settings["db_name"]]

    machine.vm.provision "shell", 
        :path => scriptDir + "/build.sh", 
        :args => [settings["sync_to"]],
        :privileged => false
end