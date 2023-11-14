
import { MySqlContainer, StartedMySqlContainer } from "@testcontainers/mysql"




export async function createMysqlDatabase(): Promise<StartedMySqlContainer> {
  //  t.Helper()
  // ctx:= context.Background()
  // schemaPath, err:= filepath.Abs("../database/schema.sql")
  // require.NoError(t, err)


  const schemaPath = "./schema.sql"

  // const schema = Bun.file(schemaPath).toString()
  const container = await new MySqlContainer().withUser("username").withUserPassword("password").withDatabase("unkey").start()
  // console.log(await container.executeQuery(schema))
  return container
  // container, err:= mysql.RunContainer(

  //    ctx,
  //    mysql.WithDatabase("unkey"),
  //    mysql.WithUsername("username"),
  //    mysql.WithPassword("password"),
  //    mysql.WithScripts(schemaPath),
  //  )
  // require.NoError(t, err)
  // t.Cleanup(func() {
  //    require.NoError(t, container.Stop(ctx, nil))
  //  })

  // return container.ConnectionString(ctx)

}
